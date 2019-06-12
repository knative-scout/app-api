package handlers


import (
	"fmt"
	"github.com/kscout/serverless-registry-api/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
)

// AppSearchHandler is used search apps and return result
// in case of an empty query, it returns all the apps in the database

type SmartSearchHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h SmartSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := r.URL.Query()
	query := vars.Get("query")

	result := smartSearchDB(query, h)

	resp := map[string][]models.App{
		"apps":result,
	}


	h.RespondJSON(w, http.StatusOK, resp)
}


func smartSearchDB(query string,  h SmartSearchHandler ) []models.App{

	// else, construct a bson query will all the required parameters and find in database
	searchBson := bson.D{}
	if len(query)>0{
		searchBson = append(searchBson, bson.E{
			"$text",
			bson.D{{"$search", query}},
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


	ret := []models.App{}  //to store all result as an array of json files
	result, err := h.MDbApps.Find(h.Ctx, searchBson , options.Find().SetProjection(projection))

	if err != nil {
		panic(fmt.Errorf("failed to retrieve data from db %s", err.Error()))
	}

	for result.Next(h.Ctx) {
		a := models.App{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("Unable to get apps from database: %s", err.Error()))
		}
		ret = append(ret,a)
	}
	return ret
}