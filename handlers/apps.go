package handlers


import (
	"fmt"
	"github.com/knative-scout/app-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"strings"
)

// AppsHandler is used search apps and return result
// in case of an empty query, it returns all the apps in the database

type AppsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := r.URL.Query()
	query := vars.Get("query")
	tags := vars.Get("tags")
	categories := vars.Get("categories")

	resp := getDataFromDB(query, tags, categories, h)

	h.RespondJSON(w, http.StatusOK, resp)
}


func getDataFromDB(query string, tags string, categories string, h AppsHandler ) []models.App{

	// if query, tags or categories are empty strings return all apps as result
	// else, construct a bson query will all the required parameters and find in database
	searchBson := bson.D{}
	if len(query)>0{
		searchBson = append(searchBson, bson.E{
			"description",
				bson.D{{"$regex", ".*"+query+".*"}},
		})
	}
	if len(tags)>0{
		tags := strings.Split(tags, ",")
		searchBson = append(searchBson, bson.E{
			"tags",
				bson.D{{"$in", tags}},
		})
	}
	if len(categories)>0{
		categories := strings.Split(categories, ",")
		searchBson = append(searchBson, bson.E{
			"categories", bson.D{
				{"$in", categories}},
		})
	}


	ret := []models.App{}  //to store all result as an array of json files
	result, err := h.MDbApps.Find(h.Ctx, searchBson)

	if err != nil {
		 panic(fmt.Errorf("failed to retrieve data from db %s", err.Error()))

	}

	for result.Next(h.Ctx) {
		a := models.App{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("readTasks: couldn't make to-do item ready for display: %s", err.Error()))
		}
		ret = append(ret,a)
	}
	return ret
}