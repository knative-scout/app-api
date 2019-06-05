package handlers


import (

	"github.com/knative-scout/app-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

// HealthHandler is used to determine if the server is running
type AppsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parametres passed in the URL
	vars := r.URL.Query()

	//h.Logger.Debugf("%#v", vars)
	query:=vars.Get("query")
	tags:= vars.Get("tags")
	categories:=vars.Get("categories")

	resp := getDataFromDB(query, tags, categories, h)

	h.RespondJSON(w, http.StatusOK, resp)
}


func getDataFromDB(query string, tags string, categories string, h AppsHandler ) []models.App{

	//if query, tags or categories are empty strings return all apps as result
	//if len(query)>0{
	//	h.Logger.Debugf("%T\n",query)
	//}
	//if len(tags)>0{
	//	h.Logger.Debugf("%T\n",tags)
	//}
	//if len(categories)>0{
	//	h.Logger.Debugf("%T\n",categories)
	//}

	//var ret []models.App
	ret := []models.App{}
	result, err := h.MDbApps.Find(h.Ctx, bson.D{{}})

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