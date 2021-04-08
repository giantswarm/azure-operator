package workerpool

import (
	"math/rand"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type Pool struct {
	jobQueue chan Job
	logger   micrologger.Logger
}

func New(size int, logger micrologger.Logger) *Pool {
	p := &Pool{
		jobQueue: make(chan Job),
		logger:   logger,
	}

	for i := 0; i < size; i++ {
		p.startWorker()
	}

	return p
}

func (p *Pool) EnqueueJob(job Job) {
	go func() {
		p.jobQueue <- job
	}()
}

func (p *Pool) Stop() {
	close(p.jobQueue)
}

func (p *Pool) startWorker() {
	go func() {
		for {
			j, open := <-p.jobQueue
			if !open {
				break
			}

			if j != nil {
				err := j.Run()
				if err != nil {
					p.logger.Log("level", "debug", "message", "job execution failed", "job_id", j.ID(), "stack", microerror.JSON(err)) // nolint: errcheck
				} else {
					if !j.Finished() {
						p.EnqueueJob(j)
					} else {
						p.logger.Log("level", "debug", "message", "job finished", "job_id", j.ID()) // nolint: errcheck
					}
				}
			}

			// Random wait time between 10 and 100 milliseconds, so we avoid
			// infinite loop with idling jobs.
			waitTime := time.Duration((rand.Intn(10) + 1) * 10) // nolint:gosec
			time.Sleep(waitTime * time.Millisecond)
		}
	}()
}
