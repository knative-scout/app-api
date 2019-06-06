package handlers


import (
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
)

type AppTagsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

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


	if len(query)>0{
		tagsBson = append(tagsBson, bson.E{
			"tags", query,
			//bson.D{{"$in", query}},
		})
	}

	result, err := h.MDbApps.Find(h.Ctx, tagsBson, options.Find().SetProjection(bson.E{"tags",1}) )
	if err != nil {
		panic(fmt.Errorf("unable to query database", err.Error()))
	}

	for result.Next(h.Ctx) {
		a := []string{}
		if err = result.Decode(&a); err != nil {
			panic(fmt.Errorf("Error in decode %s", err.Error()))
		}
		h.Logger.Debugf("take this: %#T", a)
	}



	return []string{"lala", "kaka"}
}