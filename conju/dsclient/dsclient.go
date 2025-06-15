package dsclient

import (
	"context"

	"cloud.google.com/go/datastore"
)

var dsClientKey = &struct{}{}

func FromContext(ctx context.Context) *datastore.Client {
	return ctx.Value(dsClientKey).(*datastore.Client)
}

func WrapContext(ctx context.Context, client *datastore.Client) context.Context {
	return context.WithValue(ctx, dsClientKey, client)
}
