package main

import (
	"fmt"
	"encoding/json"
	
	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration
type Config struct {
	// HTTPAddr is the HTTP server's bind address
	HTTPAddr string `default:":5000" split_words:"true"`

	// DbHost is the MongoDB server host
	DbHost string `default:"localhost" split_words:"true"`

	// DbPort is the MongoDB server port
	DbPort int `default:"27017" split_words:"true"`

	// DbUser is the MongoDB user
	DbUser string `default:"knative-scout-dev" split_words:"true"`

	// DbPassword is the MongoDB password
	DbPassword string `default:"secretpassword" split_words:"true"`

	// DbName is the database to connect to inside MongoDB
	DbName string `default:"knative-scout-app-api-dev" split_words:"true"`

	// GhToken is a GitHub API token with repository read permissions
	GhToken string `split_words:"true"`
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

	if c.GhToken != "" {
		c.GhToken = "REDACTED_NOT_EMPTY"
	}


	configBytes, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to convert configuration into JSON: %s", err.Error())
	}

	return string(configBytes), nil
}
