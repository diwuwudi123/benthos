package main

import (
	"github.com/diwuwudi123/benthos/v4/internal/serverless/lambda"

	// Import all plugins defined within the repo.
	_ "github.com/diwuwudi123/benthos/v4/public/components/all"
)

func main() {
	lambda.Run()
}
