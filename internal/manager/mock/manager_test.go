package mock_test

import (
	"github.com/diwuwudi123/benthos/v4/internal/bundle"
	"github.com/diwuwudi123/benthos/v4/internal/manager/mock"
)

var _ bundle.NewManagement = &mock.Manager{}
