package main

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration
type Config struct {
	// HTTPAddr is the HTTP server's bind address
	HTTPAddr string `default:":5000" split_words:"true"`

	// DBConnURL is the MongoDB server's connection URL
	DBConnURL string `default:"mongodb://knative-scout-dev:secretpassword@localhost:27017" split_words:"true"`
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