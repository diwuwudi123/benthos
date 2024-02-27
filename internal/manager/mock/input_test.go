package mock_test

import (
	"github.com/diwuwudi123/benthos/v4/internal/component/input"
	"github.com/diwuwudi123/benthos/v4/internal/manager/mock"
)

var _ input.Streamed = &mock.Input{}
