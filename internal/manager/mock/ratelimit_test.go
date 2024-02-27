package mock_test

import (
	"github.com/diwuwudi123/benthos/v4/internal/component/ratelimit"
	"github.com/diwuwudi123/benthos/v4/internal/manager/mock"
)

var _ ratelimit.V1 = mock.RateLimit(nil)
