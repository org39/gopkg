package db

import "context"

func doTx(txCtx context.Context, fn func(context.Context) error) (chan interface{}, chan error) {
	cpanic := make(chan interface{})
	cerr := make(chan error)

	go func() {
		// defers are execute in LIFO (last in, first out) order
		defer close(cpanic)
		defer close(cerr)

		defer func() {
			// if panic occurs
			// report panic to the caller
			if r := recover(); r != nil {
				cpanic <- r
			}
		}()

		// if callback is finished normally
		// report result to the caller
		cerr <- fn(txCtx)
	}()

	return cpanic, cerr
}
