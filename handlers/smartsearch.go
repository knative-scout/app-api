package handlers


import (
	"fmt"
	"github.com/kscout/serverless-registry-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
)

// SmartSearchHandler is used search apps and return result
// in case of an empty query, it returns all the apps in the database
type SmartSearchHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h SmartSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := r.URL.Query()
	query := vars.Get("query")
	tags := vars.Get("tags")
	categories := vars.Get("categories")

	apps, categsRes, tagsRes := smartSearchDB(query, tags, categories, h)

	resp := map[string]interface{}{
		"apps":apps,
		"categories":categsRes,
		"tags":tagsRes,
	}


	h.RespondJSON(w, http.StatusOK, resp)
}


func smartSearchDB(query string, tags string, categories string, h SmartSearchHandler ) ([]models.App, []string, []string) {

	// else, construct a bson query will all the required parameters and find in database
	searchBson := bson.D{}
	if len(query)>0{
		searchBson = append(searchBson, bson.E{
			"$text",
			bson.D{{"$search", query}},
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

	// struct to set projection in mongo
	type fields struct {
		Score bson.D `string:"score"`
	}

	projection := fields{
		Score: bson.D{{
			"$meta", "textScore",
		}},
	}


	findOptions := options.Find()
	findOptions.SetSort(projection)
	findOptions.SetProjection(projection)

	ret := []models.App{}  //to store all result as an array of json files
	categsRes := []string{}
	tagsRes := []string{}

	result, err := h.MDbApps.Find(h.Ctx, searchBson , findOptions)

	if err != nil {
		panic(fmt.Errorf("failed to retrieve data from db %s", err.Error()))
	}

	for result.Next(h.Ctx) {
		a := models.App{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("Unable to get apps from database: %s", err.Error()))
		}
		ret = append(ret,a)
		categsRes = append(categsRes, a.Categories...)
		tagsRes = append(tagsRes, a.Tags...)
	}
	return ret, removeDuplicates(categsRes), removeDuplicates(tagsRes)
}