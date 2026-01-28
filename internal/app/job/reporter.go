package job

import "context"

type ProgressReporter interface {
	Report(progress int)
}

type progressKeyType struct{}

var progressKey = progressKeyType{}

func WithProgressReporter(
	ctx context.Context,
	r ProgressReporter,
) context.Context {
	return context.WithValue(ctx, progressKey, r)
}

func GetProgressReporter(ctx context.Context) ProgressReporter {
	r, _ := ctx.Value(progressKey).(ProgressReporter)
	return r
}
