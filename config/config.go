package config

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration
type Config struct {
	// ExternalURL is the host the HTTP server can be accessed by from external users.
	// This should include any URL scheme, ports, paths, subdomains, ect.
	ExternalURL url.URL `default:"http://localhost:5000" split_words:"true" required:"true"`

	// SiteURL is the URL at which the website (github.com/kscout/kscout.io) can be accessed.
	// Must include a schema.
	SiteURL url.URL `default:"http://localhost:3000" split_words:"true" required:"true"`

	// BotAPISecret is a secret value used to authenticate with the bot API
	BotAPISecret string `split_words:"true" required:"true"`

	// BotAPIURL is the URL of the bot API.
	// Must include a schema.
	BotAPIURL url.URL `default:"http://localhost:8000" envconfig:"bot_api_url" required:"true"`

	// APIAddr is the API server's bind address
	APIAddr string `default:":5000" split_words:"true" required:"true"`

	// MetricsAddr is the metrics server's bind address
	MetricsAddr string `default:":9090" split_words:"true" required:"true"`

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

	// GhPrivateKeyPath is the path to the GitHub app's secret key
	GhPrivateKeyPath string `default:"gh.private-key.pem" split_words:"true" required:"true"`

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
	// Get
	var config Config

	if err := envconfig.Process("app", &config); err != nil {
		return nil, fmt.Errorf("error loading values from environment variables: %s",
			err.Error())
	}

	// Validate
	if len(config.ExternalURL.Scheme) == 0 {
		return nil, fmt.Errorf("ExternalURL field must have scheme")
	}

	if len(config.SiteURL.Scheme) == 0 {
		return nil, fmt.Errorf("SiteURL field must have scheme")
	}

	if len(config.BotAPIURL.Scheme) == 0 {
		return nil, fmt.Errorf("BotAPIURL field must have scheme")
	}

	return &config, nil
}

// String returns a log safe version of Config in string form. Redacts any sensative fields.
func (c Config) String() (string, error) {
	// Redact fields
	if c.DbPassword != "" {
		c.DbPassword = "REDACTED_NOT_EMPTY"
	}

	if c.GhWebhookSecret != "" {
		c.GhWebhookSecret = "REDACTED_NOT_EMPTY"
	}

	if c.BotAPISecret != "" {
		c.BotAPISecret = "REDACTED_NOT_EMPTY"
	}

	// Convert to JSON
	configBytes, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to convert configuration into JSON: %s", err.Error())
	}

	return string(configBytes), nil
}
