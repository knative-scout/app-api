package main

import (
	"context"
	"os"
	"os/signal"
	"fmt"
	"net/http"

	"github.com/knative-scout/app-api/config"
	"github.com/knative-scout/app-api/handlers"
	"github.com/knative-scout/app-api/models"
	
	"github.com/Noah-Huppert/golog"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/go-github/v25/github"
	"golang.org/x/oauth2"
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
	logger := golog.NewStdLogger("app-api")

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

	// {{{1 GitHub
	// {{{2 Create client
	logger.Debug("authenticating with GitHub API")
	
	ghTokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: cfg.GhToken,
	})
	ghTokenClient := oauth2.NewClient(ctx, ghTokenSrc)
	
	gh := github.NewClient(ghTokenClient)

	// {{{2 Ensure registry repository exists
	_, _, err = gh.Repositories.Get(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName)
	if err != nil {
		logger.Fatalf("failed to get information about serverless application "+
			"registry repository: %s", err.Error())
	}

	logger.Debug("authenticated with GitHub API")

	// {{{1 Load serverless application registry repository state if database is empty
	go func() {
		loadLogger := logger.GetChild("populate-apps-db")
 
		loadLogger.Debug("checking if Db must be populated from GitHub registry repository")
		
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
		appLoader := models.AppLoader{
			Ctx: ctx,
			Gh: gh,
			Cfg: cfg,
		}
		
		apps, err := appLoader.LoadAllAppsFromRegistry("")
		if err != nil {
			loadLogger.Fatalf("failed to load apps: %s", err.Error())
		}

		// {{{1 Insert
		insertDocs := []interface{}{}

		for _, app := range apps {
			insertDocs = append(insertDocs, *app)
		}
		
		_, err = mDbApps.InsertMany(ctx, insertDocs, nil)
		if err != nil {
			loadLogger.Fatalf("failed to insert apps into db: %s", err.Error())
		}

		loadLogger.Debugf("loaded %d app(s) into database", len(apps))
	}()

	// {{{1 Router
	baseHandler := handlers.BaseHandler{
		Ctx: ctx,
		Logger: logger.GetChild("handlers"),
		Cfg: cfg,
		MDb: mDb,
		MDbApps: mDbApps,
		MDbSubmissions: mDbSubmissions,
		Gh: gh,
	}

	router := mux.NewRouter()

	router.Handle("/health", handlers.HealthHandler{
		baseHandler.GetChild("health"),
	}).Methods("GET")

	router.Handle("/apps/id/{id}", handlers.AppByIDHandler {
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
		baseHandler.GetChild("webhook"),
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
					Handler: router,
				},
			},
		},
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to serve: %s", err.Error())
		}
	}()

	logger.Infof("started server on %s", cfg.HTTPAddr)

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Fatalf("failed to shutdown server: %s", err.Error())
	}

	logger.Info("done")
}
