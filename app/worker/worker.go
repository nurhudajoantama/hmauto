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

func (w *Worker) Go(fn func(context.Context) func() error) {
	w.Group.Go(fn(w.Ctx))
}
