package handlers


import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	//"go.mongodb.org/mongo-driver/bson"
	//"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"github.com/gorilla/mux"
	"os"
)


// AppCategoriesHandler is used to get all the categories stored in the database,
// can also be used to get categories of matched apps if query is provided
type AppsDeployHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppsDeployHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := mux.Vars(r)
	appID := vars["appID"]

	file, err := os.Open("handlers/deploy.sh")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("unable to close file : %s",err))
		}
	}()

	b, err := ioutil.ReadAll(file)
	//fmt.Print(b)

	resp := strings.Replace(string(b), "{{app.id}}", appID,1)

	h.RespondTEXT(w, http.StatusOK,resp)
}

