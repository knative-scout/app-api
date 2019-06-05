package handlers


import (

	"github.com/knative-scout/app-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"github.com/gorilla/mux"
)

// HealthHandler is used to determine if the server is running
type AppsHandler struct {
	BaseHandler
}
// ServeHTTP implements http.Handler
func (h AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parametres passed in the URL
	vars := mux.Vars(r)

	id := vars["id"]

	resp := getDataFromDB(id, h)

	h.RespondJSON(w, http.StatusOK, resp)
}


func getDataFromDB(id string, h AppsHandler ) []models.App{

	ret := []models.App{}
	result, err := h.MDbApps.Find(h.Ctx, bson.D{{"app_id" : id}})

	if err != nil {
		 h.Logger.Fatalf("failed to retrieve data from db %s", err)
	}

	for result.Next(h.Ctx) {
		a := models.App{}
		if err = result.Decode(&a); err != nil {
			h.Logger.Fatalf("readTasks: couldn't make to-do item ready for display: %v", err)
		}
		ret = append(ret,a)
	}
	return ret
}