package workerpool

import "github.com/giantswarm/micrologger"

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
			select {
			case j, closed := <-p.jobQueue:
				if j != nil {
					err := j.Run()
					if err != nil {
						p.logger.Log("foobar")
					} else {
						if !j.Finished() {
							p.EnqueueJob(j)
						}
					}
				}

				if closed {
					break
				}
			}
		}
	}()
}
