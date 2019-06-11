package handlers


import (

	"github.com/kscout/serverless-registry-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
)

// AppByIDHandler returns a single app by ID from the database
type AppByIDHandler struct {
	BaseHandler
}
// ServeHTTP implements http.Handler
func (h AppByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parametres passed in the URL
	vars := mux.Vars(r)

	id := vars["id"]

	app := getAppIDDataFromDB(id,h)

	if app != nil {
		resp :=  map[string]interface{}{"app": app}

		h.RespondJSON(w, http.StatusOK, resp) 
	} else {
		resp :=  map[string]interface{}{"error": "app not found"}
		h.RespondJSON(w,http.StatusNotFound,resp)
	}
}


func getAppIDDataFromDB(id string, h AppByIDHandler ) *models.App{

	ret := []models.App{}
	result, err := h.MDbApps.Find(h.Ctx, bson.D{{"app_id", id}})

	if err != nil {
		panic(fmt.Errorf("failed to retrieve data from db: %s", err.Error()))
	}

	for result.Next(h.Ctx) {
		a := models.App{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("readTasks: couldn't make to-do item ready for display: %s", err.Error()))
		}
		ret = append(ret,a)
	}

	if len(ret) == 0 {
		return nil
	}

	return &ret[0]
}