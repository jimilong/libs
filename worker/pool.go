package worker

import (
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Job interface {
	Do() error
	AfterDo(error)
}

type Pool struct {
	jobs      chan Job
	quit      chan struct{}
	closed    int32
	workerNum int
	wg        sync.WaitGroup
}

func NewPool(workerNum int) *Pool {
	return &Pool{
		jobs:      make(chan Job),
		quit:      make(chan struct{}),
		workerNum: workerNum,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workerNum; i++ {
		go p.worker()
	}
}

func (p *Pool) AddJob(job Job) error {
	if atomic.LoadInt32(&p.closed) != 0 {
		return errors.Errorf("worker Pool queue closed")
	}
	p.jobs <- job

	return nil
}

func (p *Pool) Stop() {
	atomic.StoreInt32(&p.closed, 1)
	close(p.quit)
	p.wg.Wait()
	close(p.jobs)
}

func (p *Pool) worker() {
	p.wg.Add(1)
	defer p.wg.Done()
	for {
		select {
		case <-p.quit:
			return
		case job := <-p.jobs:
			// 处理任务
			err := job.Do()
			job.AfterDo(err)
		}
	}
}
