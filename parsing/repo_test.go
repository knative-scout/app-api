package parsing

import (
	"testing"
	"context"

	"github.com/stretchr/testify/mock"
)

// TestGoodGetApp ensures that a correctly formatted app in the serverless registry repository is parsed correctly
func TestGoodGetApp(t *testing.T) {
	parser := RepoParser{
		Ctx: context.Background(),
}
