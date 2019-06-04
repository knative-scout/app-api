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

	// {{{1 Configuration
	config, err := NewConfig()

	if err != nil {
		logger.Fatalf("failed to load configuration: %s", err.Error())
	}

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
	mDb, err := mongo.Connect(ctx, mDbConnOpts)
	if err != nil {
		logger.Fatalf("failed to connect to database: %s", err.Error())
	}

	if err := mDb.Ping(ctx, nil); err != nil {
		logger.Fatalf("failed to test datbase connection: %s", err.Error())
	}
	
	// {{{1 Router
	baseHandler := handlers.BaseHandler{
		Ctx: ctx,
		Logger: logger.GetChild("handlers"),
		MDb: mDb,
	}

	router := mux.NewRouter()
	
	router.Handle("/health", handlers.HealthHandler{
		baseHandler.GetChild("health"),
	}).Methods("GET")

	// {{{1 Start HTTP server	
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
