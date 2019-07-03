package jobs

import (
	"fmt"
	"context"
	"strings"
	
	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/parsing"
	"github.com/kscout/serverless-registry-api/models"
	
	"github.com/google/go-github/v26/github"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

// UpdateAppsJob updates the apps collection based on the current master branch state
// The data field is unused
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

	return nil
}
