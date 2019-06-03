package handlers

import (
	"context"
	"net/http"
	"encoding/json"

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

// GetChild makes a child instance of the base handler with a prefix
func (h BaseHandler) GetChild(prefix string) BaseHandler {
	return BaseHandler{
		Ctx: h.Ctx,
		Logger: h.Logger.GetChild(prefix),
		MDb: h.MDb,
	}
}

// RepondJSON sends an object as a JSON encoded response
func (h BaseHandler) RespondJSON(w http.ResponseWriter, status int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		panic(err)
	}
}
