package worker

import (
	"context"

	"golang.org/x/sync/errgroup"
)

type Worker struct {
	Group *errgroup.Group
	Ctx   context.Context
}

func New(group *errgroup.Group, ctx context.Context) *Worker {
	return &Worker{
		Group: group,
		Ctx:   ctx,
	}
}

func (w *Worker) Go(fn func(context.Context) error) {
	w.Group.Go(func() error {
		return fn(w.Ctx)
	})
}
