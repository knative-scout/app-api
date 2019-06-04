package handlers

import (
	"fmt"
	"context"
	"net/http"
	"encoding/json"

	"github.com/Noah-Huppert/golog"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/google/go-github/v25/github"
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

	// Gh is the GitHub API client
	Gh *github.Client
}

// GetChild makes a child instance of the base handler with a prefix
func (h BaseHandler) GetChild(prefix string) BaseHandler {
	return BaseHandler{
		Ctx: h.Ctx,
		Logger: h.Logger.GetChild(prefix),
		MDb: h.MDb,
	}
}

// RespondJSON sends an object as a JSON encoded response
func (h BaseHandler) RespondJSON(w http.ResponseWriter, status int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		panic(fmt.Errorf("failed to encode response as JSON: %s", err.Error()))
	}
}

// ParseJSON parses a request body as JSON
func (h BaseHandler) ParseJSON(r *http.Request, dest interface{}) {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dest); err != nil {
		panic(fmt.Errorf("failed to decode request body as JSON: %s", err.Error()))
	}
}
