package mock_test

import (
	"github.com/diwuwudi123/benthos/v4/internal/component/cache"
	"github.com/diwuwudi123/benthos/v4/internal/manager/mock"
)

var _ cache.V1 = &mock.Cache{}
