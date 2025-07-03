package dsclient

import (
	"context"

	"cloud.google.com/go/datastore"
)

type contextKey struct{}

func FromContext(ctx context.Context) *datastore.Client {
	return ctx.Value(contextKey{}).(*datastore.Client)
}

func WrapContext(ctx context.Context, client *datastore.Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}
