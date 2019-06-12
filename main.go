package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/handlers"
	"github.com/kscout/serverless-registry-api/jobs"

	"github.com/Noah-Huppert/golog"
	"github.com/google/go-github/v26/github"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/bradleyfalzon/ghinstallation"
)

func main() {
	// {{{1 Context
	ctx, ctxCancel := context.WithCancel(context.Background())

	// signals holds signals received by process
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	go func() {
		<-signals

		ctxCancel()
	}()

	// {{{1 Logger
	logger := golog.NewStdLogger("serverless-registry-api")

	logger.Debug("starting")

	// {{{1 Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatalf("failed to load configuration: %s", err.Error())
	}

	cfgStr, err := cfg.String()
	if err != nil {
		logger.Fatalf("failed to convert configuration into string for debug log: %s",
			err.Error())
	}

	logger.Debugf("loaded configuration: %s", cfgStr)

	// {{{1 MongoDB
	// {{{2 Build connection options
	mDbConnOpts := options.Client()
	mDbConnOpts.SetAuth(options.Credential{
		Username: cfg.DbUser,
		Password: cfg.DbPassword,
	})
	mDbConnOpts.SetHosts([]string{
		fmt.Sprintf("%s:%d", cfg.DbHost, cfg.DbPort),
	})

	if err = mDbConnOpts.Validate(); err != nil {
		logger.Fatalf("failed to validate database connection options: %s", err.Error())
	}

	// {{{2 Connect
	logger.Debug("connecting to Db")

	mDbClient, err := mongo.Connect(ctx, mDbConnOpts)
	if err != nil {
		logger.Fatalf("failed to connect to database: %s", err.Error())
	}

	if err := mDbClient.Ping(ctx, nil); err != nil {
		logger.Fatalf("failed to test database connection: %s", err.Error())
	}

	mDb := mDbClient.Database(cfg.DbName)
	mDbApps := mDb.Collection("apps")
	mDbSubmissions := mDb.Collection("submissions")

	logger.Debug("connected to Db")

	// {{{2 Add indexes if none found
	mDbAppsIndexes := mDbApps.Indexes()
	
	indexesCur, err := mDbAppsIndexes.List(ctx, nil)
	if err != nil {
		logger.Fatalf("failed to list db indexes: %s", err.Error())
	}

	indexCount := 0
	for indexesCur.Next(ctx) {
		indexCount++
	}

	if indexCount == 1 {
		_, err := mDbAppsIndexes.CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{{"$**", "text"}},
		})
		if err != nil {
			logger.Fatalf("failed to create db index: %s", err.Error())
		}

		logger.Debug("created db indexes")
	} else {
		logger.Debugf("db already has %d indexes", indexCount)
	}

	// {{{1 GitHub
	// {{{2 Create client
	logger.Debug("authenticating with GitHub API")

	ghAPIKeyTransport, err := ghinstallation.NewKeyFromFile(http.DefaultTransport,
		cfg.GhIntegrationID, cfg.GhInstallationID, cfg.GhPrivateKeyPath)
	if err != nil {
		logger.Fatalf("failed to load GitHub API secret key file: %s", err.Error())
	}

	gh := github.NewClient(&http.Client{Transport: ghAPIKeyTransport})

	// {{{2 Ensure registry repository exists
	_, _, err = gh.Repositories.Get(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName)
	if err != nil {
		logger.Fatalf("failed to get information about serverless application "+
			"registry repository: %s", err.Error())
	}

	logger.Debug("authenticated with GitHub API")

	// {{{1 Setup component shutdown bus
	// targetShutdownBusCount is the number of messages which must be received on
	// shutdownBus before the process will exit gracefully
	targetShutdownBusCount := 2
	
	// shutdownBus receives a message with a component's name when a component shuts down.
	// This lets the process wait until all of its components have been shut down gracefully
	// before exiting.
	//
	// Currently the process has the following components:
	//
	//    - (job-runner): Job runner
	//    - (http-api) HTTP API server
	// 
	// The end of the program will wait for targetShutdownBusCount number of messages to
	// be sent over this bus before exiting. Each of the components above should send a
	// message on the bus when they are done.
	shutdownBus := make(chan string, targetShutdownBusCount)

	// {{{1 Pull request evalator
	jobRunner := &jobs.JobRunner{
		Ctx: ctx,
		Logger: logger.GetChild("job-runner"),
		Cfg: cfg,
		GH: gh,
		MDbApps: mDbApps,
	}
	jobRunner.Init()

	go func() {
		jobRunner.Run()

		shutdownBus <- "job-runner"
	}()

	// {{{1 Load applications from database if none exist yet
	go func() {
		loadLogger := logger.GetChild("populate-apps-db")

		// {{{1 Check if empty
		docCount, err := mDbApps.CountDocuments(ctx, bson.D{{}}, nil)
		if err != nil {
			loadLogger.Fatalf("failed to get documents count in apps collection: %s",
				err.Error())
		}

		if docCount > 0 {
			loadLogger.Debugf("no load required, found %d app(s) in database",
				docCount)
			return
		}

		// {{{1 Load all apps if empty
		loadLogger.Debugf("no apps found, will load apps into database")
		
		jobRunner.Submit(jobs.JobStartRequest{
			Type: jobs.JobTypeUpdateApps,
		})
	}()

	// {{{1 Router
	baseHandler := handlers.BaseHandler{
		Ctx:            ctx,
		Logger:         logger.GetChild("handlers"),
		Cfg:            cfg,
		MDb:            mDb,
		MDbApps:        mDbApps,
		MDbSubmissions: mDbSubmissions,
		Gh:             gh,
	}

	router := mux.NewRouter()

	router.Handle("/health", handlers.HealthHandler{
		baseHandler.GetChild("health"),
	}).Methods("GET")

	router.Handle("/apps/id/{id}", handlers.AppByIDHandler{
		baseHandler.GetChild("get-app-by-id"),
	}).Methods("GET")

	router.Handle("/apps", handlers.AppSearchHandler{
		baseHandler.GetChild("app-search"),
	}).Methods("GET")

	router.Handle("/apps/tags", handlers.AppTagsHandler{
		baseHandler.GetChild("get-apps-tags"),
	}).Methods("GET")

	router.Handle("/apps/categories", handlers.AppCategoriesHandler{
		baseHandler.GetChild("get-apps-categories"),
	}).Methods("GET")

	router.Handle("/apps/webhook", handlers.WebhookHandler{
		BaseHandler: baseHandler.GetChild("webhook"),
		JobRunner: jobRunner,
	}).Methods("POST")

	// !!! Must always be last !!!
	router.Handle("/", handlers.PreFlightOptionsHandler{
		baseHandler.GetChild("pre-flight-options"),
	}).Methods("OPTIONS")

	// {{{1 Start HTTP server
	logger.Debug("starting HTTP server")

	server := http.Server{
		Addr: cfg.HTTPAddr,
		Handler: handlers.PanicHandler{
			BaseHandler: baseHandler,
			Handler: handlers.ReqLoggerHandler{
				BaseHandler: baseHandler,
				Handler: handlers.CORSHandler{
					BaseHandler: baseHandler,
					Handler:     router,
				},
			},
		},
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to serve: %s", err.Error())
		}

		shutdownBus <- "http-api"
	}()

	logger.Infof("started server on %s", cfg.HTTPAddr)

	// {{{1 Wait for all components to shut down
	<-ctx.Done()

	logger.Infof("shutting down %d components", targetShutdownBusCount)

	go func() {
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Fatalf("failed to shutdown server: %s",
				err.Error())
		}
	}()
	
	shutdownBusRecvCount := 0

	for shutdownBusRecvCount < targetShutdownBusCount {
		name := <-shutdownBus
		shutdownBusRecvCount++
		logger.Infof("%s component shut down (%d/%d)", name, shutdownBusRecvCount, targetShutdownBusCount)
	}

	logger.Info("done")
}
