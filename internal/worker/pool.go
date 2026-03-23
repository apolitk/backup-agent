package worker

import (
	"context"
	"sync"
)

type Task func(context.Context) error

type Pool struct {
	workers int
	queue   chan Task
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func New(workers int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		workers: workers,
		queue:   make(chan Task, workers*2),
		ctx:     ctx,
		cancel:  cancel,
	}
	p.start()
	return p
}

func (p *Pool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.queue:
			if !ok {
				return
			}
			task(p.ctx)
		}
	}
}

func (p *Pool) Submit(task Task) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.queue <- task:
		return nil
	}
}

func (p *Pool) Stop() {
	p.cancel()
	close(p.queue)
	p.wg.Wait()
}