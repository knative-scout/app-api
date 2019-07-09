package jobs

import (
	"fmt"
	"context"
	"strings"
	"net/http"
	"bytes"
	"encoding/json"
	"io/ioutil"
	
	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/parsing"
	"github.com/kscout/serverless-registry-api/models"
	"github.com/kscout/serverless-registry-api/req"
	
	"github.com/google/go-github/v26/github"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

// UpdateAppsJobDefinition specifies the behavior of an UpdateAppsJob
type UpdateAppsJobDefinition struct {
	// NoBotAPINotify when true indicates that the job should not make a request 
	// to the bot API new apps endpoint
	NoBotAPINotify bool
}

// UpdateAppsJob updates the apps collection based on the current master branch state
// The data field is optional. If provided must be a JSON encoded UpdateAppsJobDefinition.
type UpdateAppsJob struct {
	// Ctx
	Ctx context.Context
	
	// Cfg is the server configuration
	Cfg *config.Config
	
	// GH is a GitHub API client
	GH *github.Client

	// MDbApps is used to access the apps collection
	MDbApps *mongo.Collection
}

// Do job actions
func (j UpdateAppsJob) Do(data []byte) error {
	// {{{1 Parse data field if provided
	var jobDef UpdateAppsJobDefinition
	
	if len(data) > 0 {
		if err := json.Unmarshal(data, &jobDef); err != nil {
			return fmt.Errorf("failed to decode data field as "+
				"UpdateAppsJobDefinition JSON: %s", err.Error())
		}
	}
	
	// {{{1 Get all apps in registry repository
	repoParser := parsing.RepoParser{
		Ctx: j.Ctx,
		GH: j.GH,
		GHDevTeamName: j.Cfg.GhDevTeamName,
		SiteURL: j.Cfg.SiteURL,
		RepoOwner: j.Cfg.GhRegistryRepoOwner,
		RepoName: j.Cfg.GhRegistryRepoName,
		RepoRef: "master",
	}
	appIDs, err := repoParser.GetAppIDs()
	if err != nil {
		return fmt.Errorf("failed to get IDs of application in repository: %s",
			err.Error())
	}

	apps := map[string]models.App{}

	for _, appID := range appIDs {
		app, errs := repoParser.GetApp(appID)
		if errs != nil {
			errStrs := []string{}
			for _, err := range errs {
				errStrs = append(errStrs, err.Error())
			}
			return fmt.Errorf("failed to get application with ID %s: %s",
				appID, strings.Join(errStrs, ", "))
		}

		apps[appID] = *app
	}

	// {{{1 Save in database
	upsertTrue := true
	for appID, app := range apps {
		_, err := j.MDbApps.UpdateOne(j.Ctx, bson.D{{"app_id", appID}},
			bson.D{{"$set", app}}, &options.UpdateOptions{
				Upsert: &upsertTrue,
			})
		if err != nil {
			return fmt.Errorf("failed to update app with ID %s in db: %s",
				app.AppID, err.Error())
		}
	}

	// {{{1 Delete any old apps
	_, err = j.MDbApps.DeleteMany(j.Ctx, bson.D{{
		"app_id",
		bson.D{{
			"$nin",
			appIDs,
		}},
	}}, nil)
	if err != nil {
		return fmt.Errorf("failed to prune old apps from db: %s", err.Error())
	}

	// {{{1 Notify bot API of data change
	// {{{2 Do not notify if UpdateAppsJobDefinition.NoBotAPINotify field is set
	if jobDef.NoBotAPINotify {
		return nil
	}

	// {{{2 Setup request
	// {{{3 URL
	reqURL := j.Cfg.BotAPIURL
	reqURL.Path = "/newapps"

	// {{{3 Body
	appsValues := []models.App{}
	for _, app := range apps {
		appsValues = append(appsValues, app)
	}

	reqBuf := bytes.NewBuffer(nil)
	reqEncoder := json.NewEncoder(reqBuf)

	reqBody := map[string]interface{}{
		"apps": appsValues,
	}

	if err := reqEncoder.Encode(reqBody); err != nil {
		return fmt.Errorf("failed to encode apps array as JSON: %s", err.Error())
	}

	reqReadCloser := req.ReaderDummyCloser{
		reqBuf,
	}

	// {{{3 Actual request
	req := http.Request{
		Method: "POST",
		URL: &reqURL,
		Header: map[string][]string{
			"Content-Type": {"application/json"},
			"Authorization": {j.Cfg.BotAPISecret},
		},
		Body: reqReadCloser,
	}

	// {{{2 Make request
	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return fmt.Errorf("failed to make new apps request to bot API: %s",
			err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("got non-OK response from new apps endpoint for "+
				"bot API but failed to read response body, status: %s, "+
				"body read error: %s",
				resp.Status, err.Error())
		}
		return fmt.Errorf("got non-OK response from new apps endpoint for bot API "+
			"status: %s, body: %s", resp.Status, string(respBody))
	}

	return nil
}
