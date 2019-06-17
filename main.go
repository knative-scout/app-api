package main

import (
	"strconv"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"flag"
	"path/filepath"
	"encoding/json"

	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/handlers"
	"github.com/kscout/serverless-registry-api/models"
	"github.com/kscout/serverless-registry-api/jobs"
	"github.com/kscout/serverless-registry-api/validation"

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

	// {{{2 Ensure database indexes exist
	mDbAppsIndexes := mDbApps.Indexes()
	
	_, err = mDbAppsIndexes.CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"$**", "text"}},
	})
	if err != nil {
		logger.Fatalf("failed to create db index: %s", err.Error())
	}

	logger.Debugf("ensured db indexes exist")

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

	// {{{1 Job runner
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

	// {{{1 Run quick script actions
	// {{{2 Parse flags
	// Flags can be provided which make the main.go file act more as a script.
	// If a flag is provided the server will perform a specific action and then exit.

	// doUpdateJob indicates if the server should submit an update job and then exit
	var doUpdateJob bool

	// doSeed indicates that the server should import seed data into the datbase and exit
	var doSeed bool

	// doValidatePRNum indicates that the server should run a validate job for a PR with the
	// specified num
	var doValidatePRNum string

	flag.BoolVar(&doUpdateJob, "update-apps", false,
		"If provided server will run one update job and exit. Must be only "+
			"flag provided.")
	flag.BoolVar(&doSeed, "seed", false,
		"If provided server will import seed data from the ./seed-data folder. This "+
			"folder should hold JSON files which contain 1 app each. Must be "+
			"the only flag provided")
	flag.StringVar(&doValidatePRNum, "validate-pr", "",
		"If provided will run a validate job for the GitHub pull request with the "+
			"provided number. Must be the only flag provided")
	flag.Parse()

	// {{{2 Do actions
	if doUpdateJob {
		logger.Info("running UpdateApps job and then exiting")
		req := jobRunner.Submit(jobs.JobTypeUpdateApps, nil)
		<-req.CompleteChan
		os.Exit(0)
	} else if doSeed {
		logger.Info("reading seed data")

		// for each file in the seed data directory
		err := filepath.Walk("./seed-data",
			func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				
				if info.IsDir() {
					return nil
				}
				
				// Open file
				f, err := os.Open(p)
				if err != nil {
					return fmt.Errorf("failed to open file: %s", err.Error())
				}

				// Decode file into App
				decoder := json.NewDecoder(f)

				var app models.App
				if err := decoder.Decode(&app); err != nil {
					return fmt.Errorf("failed to decode JSON file "+
						"into app: %s", err.Error())
				}

				// Validate
				if err := validation.ValidateApp(app); err != nil {
					return fmt.Errorf("failed to validate file: %s",
						err.Error())
				}

				// Save into database
				logger.Debugf("seeding %#v into db", app)
				
				upsertTrue := true
				_, err = mDbApps.UpdateOne(ctx,
					bson.D{{"app_id", app.AppID}},
					bson.D{{"$set", app}},
					&options.UpdateOptions{
						Upsert: &upsertTrue,
					})
				if err != nil {
					return fmt.Errorf("failed to update app with ID %s "+
						"in db: %s", app.AppID, err.Error())
				}

				return nil
			})
		if err != nil {
			logger.Fatalf("failed to seed database: %s", err.Error())
		}

		os.Exit(1)
	} else if len(doValidatePRNum) > 0 {
		logger.Infof("will launch validate job for PR #%s", doValidatePRNum)

		// Convert pr number into integer
		prNum, err := strconv.Atoi(doValidatePRNum)
		if err != nil {
			logger.Fatalf("failed to convert specified PR number string to int: %s",
				err.Error())
		}

		// Get PR
		pr, _, err := gh.PullRequests.Get(ctx, cfg.GhRegistryRepoOwner,
			cfg.GhRegistryRepoName, prNum)
		if err != nil {
			logger.Fatalf("failed to get pull request with number %d: %s",
				prNum, err.Error())
		}

		// Convert PR to bytes
		prBytes, err := json.Marshal(pr)
		if err != nil {
			logger.Fatalf("failed to marshal PR into JSON: %s", err.Error())
		}
		req := jobRunner.Submit(jobs.JobTypeValidate, prBytes)
		<-req.CompleteChan
		os.Exit(0)
	}

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
		
		jobRunner.Submit(jobs.JobTypeUpdateApps, nil)
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

	router.Handle("/nsearch", handlers.SmartSearchHandler{
		baseHandler.GetChild("nsearch"),
	}).Methods("GET")

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
