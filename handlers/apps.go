package handlers


import (
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

	var resp = getDataFromDB(query, tags, categories)

	h.RespondJSON(w, http.StatusOK, resp)
}


func getDataFromDB(query string, tags string, categories string) map[string]string{

	//if query, tags or categories are empty strings return all apps as result
	if len(query)>0{
		h.Logger.Debugf("%T\n",query)
	}
	if len(tags)>0{
		h.Logger.Debugf("%T\n",tags)
	}
	if len(categories)>0{
		h.Logger.Debugf("%T\n",categories)
	}



	return map[string]string{
		"ok": "true",
	}

}