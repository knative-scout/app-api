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


// AppsDeployHandler is used to send deploy.sh file in curl to users' terminal
type AppsDeployHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h AppsDeployHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//gets all the optional parameters passed in the URL
	vars := mux.Vars(r)
	appID := vars["appID"]

	//opening deploy.sh file
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

	resp := strings.Replace(string(b), "{{app.id}}", appID,-1)

	h.RespondTEXT(w, http.StatusOK,resp)
}

