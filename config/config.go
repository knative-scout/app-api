package config

import (
	"fmt"
	"encoding/json"
	
	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration
type Config struct {
	// HTTPAddr is the HTTP server's bind address
	HTTPAddr string `default:":5000" split_words:"true" required:"true"`

	// DbHost is the MongoDB server host
	DbHost string `default:"localhost" split_words:"true" required:"true"`

	// DbPort is the MongoDB server port
	DbPort int `default:"27017" split_words:"true" required:"true"`

	// DbUser is the MongoDB user
	DbUser string `default:"kscout-dev" split_words:"true" required:"true"`

	// DbPassword is the MongoDB password
	DbPassword string `default:"secretpassword" split_words:"true" required:"true"`

	// DbName is the database to connect to inside MongoDB
	DbName string `default:"kscout-serverless-registry-api-dev" split_words:"true" required:"true"`

	// GhSecretKeyPath is the path to the GitHub app's secret key
	GhSecretKeyPath string `default:"gh.private-key.pem" split_words:"true" required:"true"`

	// GhIntegrationID is the Scout Bot GitHub App ID
	GhIntegrationID int `split_words:"true" required:"true"`

	// GhInstallationID is an ID sent in the Scout Bot GitHub App webhook
	GhInstallationID int `split_words:"true" required:"true"`

	// GhRegistryRepoOwner is the GitHub user / organization which owns the serverless
	// application registry repository.
	GhRegistryRepoOwner string `default:"kscout" split_words:"true" required:"true"`

	// GhRegistryRepoName is the name of the GitHub repository which acts as a serverless
	// application registry.
	GhRegistryRepoName string `default:"serverless-apps" split_words:"true" required:"true"`

	// GhWebhookSecret is the secret token used to verify requests to the Webhook came
	// from GitHub
	GhWebhookSecret string `split_words:"true" required:"true"`

	// GhDevTeamName is the name of an organization team on GitHub which should be pinged
	// by pull request bot if any internal server errors occur
	GhDevTeamName string `default:"@kscout/developers" split_words:"true" required:"true"`
}

// NewConfig loads configuration values from environment variables
func NewConfig() (*Config, error) {
	var config Config

	if err := envconfig.Process("app", &config); err != nil {
		return nil, fmt.Errorf("error loading values from environment variables: %s",
			err.Error())
	}

	return &config, nil
}

// String returns a log safe version of Config in string form. Redacts any sensative fields.
func (c Config) String() (string, error) {
	if c.DbPassword != "" {
		c.DbPassword = "REDACTED_NOT_EMPTY"
	}

	if c.GhWebhookSecret != "" {
		c.GhWebhookSecret = "REDACTED_NOT_EMPTY"
	}

	configBytes, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to convert configuration into JSON: %s", err.Error())
	}

	return string(configBytes), nil
}
