package risk

import (
	"context"
	"time"
)

func timeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
