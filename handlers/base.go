package handlers

import (
	"context"

	"github.com/Noah-Huppert/golog"
	"go.mongodb.org/mongo-driver/mongo"
)

// BaseHandler provides helper methods and commonly used variables for API endpoints to base
// their http.Handlers off
type BaseHandler struct {
	// Ctx is the application context
	Ctx context.Context

	// Logger logs information
	Logger golog.Logger

	// MDb is the MongoDB client
	MDb *mongo.Client
}

