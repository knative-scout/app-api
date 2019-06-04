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


	vars := r.URL.Query()

	h.Logger.Debugf("%#v", vars)
	query:=vars.Get("query")
	tags:= vars.Get("tags")
	categories:=vars.Get("categories")
	h.Logger.Debug(query)
	h.Logger.Debug(tags)
	h.Logger.Debug(categories)
	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
