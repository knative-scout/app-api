package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaRequiredValidation(t *testing.T) {
	// Set env vars which need to have a non-nil value
	for _, key := range []string{"APP_BOT_API_SECRET", "APP_GH_INTEGRATION_ID", "APP_GH_INSTALLATION_ID", "APP_GH_WEBHOOK_SECRET"} {
		err := os.Setenv(key, "123") // Set env var to a number so fields which require a number don't complain

		assert.NoErrorf(t, err, "failed to set \"%s\" key to a non-empty value", key)
	}

	// Set URLs to have no schema
	for _, key := range []string{"APP_EXTERNAL_URL", "APP_SITE_URL", "APP_BOT_API_URL"} {
		err := os.Setenv(key, key)

		assert.NoErrorf(t, err, "failed to set \"%s\" key to a URL with no scheme "+
			"for test", key)
	}

	_, err := NewConfig()
	assert.NotNil(t, err, "NewConfig should have responded with an error")

	// Set URLs to have a schema
	for _, key := range []string{"APP_EXTERNAL_URL", "APP_SITE_URL", "APP_BOT_API_URL"} {
		err := os.Setenv(key, fmt.Sprintf("https://%s", key))

		assert.NoErrorf(t, err, "failed to set \"%s\" key to a URL with a scheme "+
			"for test", key)
	}

	_, err = NewConfig()
	assert.Nil(t, err, "NewConfig should have responded with no error")
}
