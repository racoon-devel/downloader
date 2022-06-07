package job

import (
	"context"
	"sync"
)

type Job = func()

type Pool struct {
	workers sync.WaitGroup
	jobs    sync.WaitGroup
	ch      chan Job
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPool(workers uint, maxBufferingJobs uint) *Pool {
	pool := &Pool{
		ch: make(chan Job, maxBufferingJobs),
	}

	pool.ctx, pool.cancel = context.WithCancel(context.Background())

	for i := uint(0); i < workers; i++ {
		pool.workers.Add(1)
		go func() {
			defer pool.workers.Done()
			for {
				select {
				case job := <-pool.ch:
					job()
					pool.jobs.Done()

				case <-pool.ctx.Done():
					return
				}

			}
		}()
	}

	return pool
}

func (pool *Pool) Run(job Job) {
	pool.jobs.Add(1)
	pool.ch <- job
}

func (pool *Pool) Wait() {
	pool.jobs.Wait()
}

func (pool *Pool) Close() {
	pool.cancel()
	pool.workers.Wait()
	close(pool.ch)
}
