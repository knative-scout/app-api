package handlers


import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"

	//"strings"
)


// AppTagsHandler is used to get all the tags stored in the database,
// can also be used to get tags of matched apps if query is provided
type AppTagsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := r.URL.Query()
	query := vars.Get("query")

	result := getTagsFromDB(query, h)

	resp := map[string][]string{
		"tags":result,
	}

	h.RespondJSON(w, http.StatusOK, resp)
}


func getTagsFromDB(query string, h AppTagsHandler ) []string {

	tagsBson := bson.D{}

	// if we have additional parametres, only get the tags of matched apps.
	if len(query)>0{
		query := strings.Split(query, ",")
		tagsBson = append(tagsBson, bson.E{
			"tags",
			bson.D{{"$in", query}},
		})
	}

	// Declaring structure to set projection in find. used to single out the column. for more info see *findoptions in mongo
	type fields struct {
		Tags int `bson:"tags"`
	}

	// Declaring structure to decode the output from the mongo tags search query
	type tagsRes struct{
		Tags []string `json:"tags" bson:"tags"`
	}

	// setting projection
	projection := fields{
		Tags: 1,
	}

	result, err := h.MDbApps.Find(h.Ctx, tagsBson, options.Find().SetProjection(projection) )
	if err != nil {
		panic(fmt.Errorf("unable to query database", err.Error()))
	}

	// declaring return string
	ret := []string{}

	for result.Next(h.Ctx) {
		a := tagsRes{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("Error in decode %s", err.Error()))
		}
		ret= append(ret, a.Tags...)

	}

	return ret
}