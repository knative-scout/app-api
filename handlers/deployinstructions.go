package handlers

import (
	"fmt"
	"net/http"

	"github.com/kscout/serverless-registry-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"github.com/gorilla/mux"
)

// DeployInstructionsHandler provides user with deployment instructions
type DeployInstructionsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler.ServerHTTP
func (h DeployInstructionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// {{{1 Try to find app with ID
	vars := mux.Vars(r)

	id := vars["id"]

	res := h.MDbApps.FindOne(h.Ctx, bson.D{{"app_id", id}})
	if res.Err() != nil {
		panic(fmt.Errorf("failed to query database for app with ID: %s",
			res.Err().Error()))
	}

	var app models.App
	if err := res.Decode(&app); err == mongo.ErrNoDocuments {
		h.RespondJSON(w, http.StatusNotFound, map[string]string{
			"error": "app not found",
		})
		return
	}

	// {{{1 Return instructions
	h.RespondJSON(w, http.StatusOK, map[string]string{
		"instructions": fmt.Sprintf("To deploy the %s application run the following command **in bash**:  \n"+
			"```\n"+
			". <(curl -L %s/apps/id/%s/deploy.sh)\n"+
			"```\n"+
			"  \n"+
			"This command will complete the following actions:\n"+
			"- Present values for any `ConfigMap` and / or `Secret` resources and give you a chance to customize them\n"+
			"- Deploy app's resource manifests, with any custom values, to your cluster using `kubectl`\n"+
			"  \n"+
			"You can see the [raw resource manifests for the %s app here](%s/apps/id/%s/deployment.json)",
			app.Name,
			h.Cfg.ExternalURL.String(), app.AppID, app.Name, h.Cfg.ExternalURL.String(), app.AppID),
	})
}
