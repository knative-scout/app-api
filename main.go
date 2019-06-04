package main

import (
	"context"
	"os"
	"os/signal"
	"fmt"
	"net/http"

	"github.com/knative-scout/app-api/handlers"
	
	"github.com/Noah-Huppert/golog"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	config, err := NewConfig()
	if err != nil {
		logger.Fatalf("failed to load configuration: %s", err.Error())
	}

	configStr, err := config.String()
	if err != nil {
		logger.Fatalf("failed to convert configuration into string for debug log: %s",
			err.Error())
	}

	logger.Debugf("loaded configuration: %s", configStr)

	// {{{1 MongoDB
	// {{{2 Build connection options
	mDbConnOpts := options.Client()
	mDbConnOpts.SetAuth(options.Credential{
		Username: config.DbUser,
		Password: config.DbPassword,
	})
	mDbConnOpts.SetHosts([]string{
		fmt.Sprintf("%s:%d", config.DbHost, config.DbPort),
	})

	if err = mDbConnOpts.Validate(); err != nil {
		logger.Fatalf("failed to validate database connection options: %s", err.Error())
	}

	// {{{2 Connect
	logger.Debug("connecting to DB")
	
	mDb, err := mongo.Connect(ctx, mDbConnOpts)
	if err != nil {
		logger.Fatalf("failed to connect to database: %s", err.Error())
	}

	if err := mDb.Ping(ctx, nil); err != nil {
		logger.Fatalf("failed to test database connection: %s", err.Error())
	}

	logger.Debug("connected to Db")

	// {{{1 GitHub
	// {{{2 Create client
	logger.Debug("authenticating with GitHub API")
	
	ghTokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: config.GhToken,
	})
	ghTokenClient := oauth2.NewClient(ctx, ghTokenSrc)
	
	gh := github.NewClient(ghTokenClient)

	// {{{2 Ensure registry repository exists
	_, _, err = gh.Repositories.Get(ctx, config.GhRegistryRepoOwner,
		config.GhRegistryRepoName)
	if err != nil {
		logger.Fatalf("failed to get information about serverless application "+
			"registry repository: %s", err.Error())
	}

	logger.Debug("authenticated with GitHub API")
	
	// {{{1 Router
	baseHandler := handlers.BaseHandler{
		Ctx: ctx,
		Logger: logger.GetChild("handlers"),
		MDb: mDb,
		Gh: gh,
	}

	router := mux.NewRouter()
	
	router.Handle("/health", handlers.HealthHandler{
		baseHandler.GetChild("health"),
	}).Methods("GET")

	// {{{1 Start HTTP server
	logger.Debug("starting HTTP server")
	
	server := http.Server{
		Addr: config.HTTPAddr,
		Handler: handlers.PanicHandler{
			BaseHandler: baseHandler,
			Handler: handlers.ReqLoggerHandler{
				BaseHandler: baseHandler,
				Handler: router,
			},
		},
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to serve: %s", err.Error())
		}
	}()

	logger.Infof("started server on %s", config.HTTPAddr)

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		logger.Fatalf("failed to shutdown server: %s", err.Error())
	}

	logger.Info("done")
}
