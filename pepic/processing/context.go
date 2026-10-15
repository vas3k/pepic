package processing

import (
	"context"
	"time"

	"github.com/vas3k/pepic/pepic/config"
)

type jobContextKey struct{}

func WithJobContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, jobContextKey{}, true)
}

func operationTimeout(ctx context.Context, seconds int) (context.Context, context.CancelFunc) {
	if background, _ := ctx.Value(jobContextKey{}).(bool); background {
		seconds = config.App.Jobs.ProcessingTimeoutSeconds
	}
	return context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
}
