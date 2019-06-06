package handlers


import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
)


// AppCategoriesHandler is used to get all the categories stored in the database,
// can also be used to get categories of matched apps if query is provided
type AppCategoriesHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppCategoriesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := r.URL.Query()
	query := vars.Get("query")

	result := getCategoriesFromDB(query, h)

	resp := map[string][]string{
		"categories":result,
	}

	h.RespondJSON(w, http.StatusOK, resp)
}


func getCategoriesFromDB(query string, h AppCategoriesHandler) []string {

	categoriesBson := bson.D{}

	// if we have additional parametres, only get the tags of matched apps.
	if len(query)>0{
		query := strings.Split(query, ",")
		categoriesBson = append(categoriesBson, bson.E{
			"categories",
			bson.D{{"$in", query}},
		})
	}

	// Declaring structure to set projection in find. used to single out the column. for more info see *findoptions in mongo
	type fields struct {
		Categories int `bson:"categories"`
	}

	// Declaring structure to decode the output from the mongo categories search query
	type categoriesRes struct{
		Categories []string `json:"categories" bson:"categories"`
	}

	// setting projection
	projection := fields{
		Categories: 1,
	}

	result, err := h.MDbApps.Find(h.Ctx, categoriesBson, options.Find().SetProjection(projection) )
	if err != nil {
		panic(fmt.Errorf("unable to query database", err.Error()))
	}

	// declaring return string
	ret := []string{}

	for result.Next(h.Ctx) {
		a := categoriesRes{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("Error in decode %s", err.Error()))
		}
		ret= append(ret, a.Categories...)

	}

	return removeDuplicates(ret)
}



