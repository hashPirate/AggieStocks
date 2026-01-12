package processor

import (
	"context"

	"eon/platform/internal/event"
)

type Handler interface {
	Handle(ctx context.Context,ev event.Event) error
}

type HandlerFunc func(ctx context.Context,ev event.Event) error

func (f HandlerFunc) Handle(ctx context.Context,ev event.Event) error {
	return f(ctx,ev)
}
