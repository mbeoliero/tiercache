package rediscache

import "context"

type Logger interface {
	CtxInfo(context.Context, string, ...interface{})
	CtxError(context.Context, string, ...interface{})
	CtxDebug(context.Context, string, ...interface{})
}
