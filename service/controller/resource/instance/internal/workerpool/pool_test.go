package workerpool

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
)

type testJob struct {
	f func() error
}

func (tj *testJob) Run() error {
	return tj.f()
}

func (tj *testJob) Finished() bool {
	return true
}

func Test_Basic_Smoke_of_WorkerPool(t *testing.T) {
	testCases := []struct {
		name       string
		testFunc   func() error
		numJobs    int
		numWorkers int
	}{
		{
			name:       "case 0: single job, single worker",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    1,
			numWorkers: 1,
		},
		{
			name:       "case 1: three jobs, single worker",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    3,
			numWorkers: 1,
		},
		{
			name:       "case 2: single job, three workers",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    1,
			numWorkers: 3,
		},
		{
			name:       "case 3: three jobs, three workers",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    3,
			numWorkers: 3,
		},
		{
			name:       "case 4: nine jobs, three workers",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    9,
			numWorkers: 3,
		},
		{
			name:       "case 5: 32 jobs, three workers",
			testFunc:   func() error { fmt.Println("hello"); return nil },
			numJobs:    32,
			numWorkers: 3,
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			wg := new(sync.WaitGroup)

			workerPool := New(tc.numWorkers, microloggertest.New())
			for i := 0; i < tc.numJobs; i++ {
				j := &testJob{
					f: func() error {
						defer wg.Done()
						return tc.testFunc()
					},
				}
				wg.Add(1)
				workerPool.EnqueueJob(j)
			}

			wg.Wait()

			workerPool.Stop()
		})
	}
}
