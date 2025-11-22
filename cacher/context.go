package cacher

import "context"

type contextKey struct{}

var runInfoKey = contextKey{}

type runInfo struct {
	level int
}

func (r *runInfo) Level() int {
	return r.level
}

func NewRunInfo(level int) RunInfo {
	return &runInfo{level: level}
}

func NewContext(ctx context.Context, info RunInfo) context.Context {
	return context.WithValue(ctx, runInfoKey, info)
}

func GetRunInfo(ctx context.Context) RunInfo {
	if info, ok := ctx.Value(runInfoKey).(RunInfo); ok {
		return info
	}
	return &runInfo{}
}
