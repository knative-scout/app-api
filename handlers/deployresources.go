package handlers


import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"strings"

	//"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/kscout/serverless-registry-api/models"
)


// AppsDeployHandler is used to send deploy.sh file in curl to users' terminal
type AppsDeployResourcesHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppsDeployResourcesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := mux.Vars(r)
	appID := vars["appID"]

	searchBson := bson.D{}

	searchBson = append(searchBson, bson.E{
		"app_id", appID ,
	})


	result := h.MDbApps.FindOne(h.Ctx, bson.D{{"app_id", appID}})
	if result.Err() != nil {
		panic(fmt.Errorf("unable to query database", result.Err().Error()))
	}


	// declaring return string
	ret := ""

	a := models.App{}
	if err := result.Decode(&a); err != nil {
		panic(fmt.Errorf("Error in decode %s", err.Error()))
	}
	ret = strings.Join(a.Deployment.Resources, "\n")

	h.RespondTEXT(w, http.StatusOK, ret)
}

