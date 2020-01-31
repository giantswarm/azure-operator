package collector

import (
	"context"
	"sync"
	"testing"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/prometheus/client_golang/prometheus"
)

func Test_Collector_Set_Boot(t *testing.T) {
	for i := 0; i < 100; i++ {
		s, err := newTestSet()
		if err != nil {
			t.Fatalf(err.Error())
		}

		var wg sync.WaitGroup

		for g := 0; g < 100; g++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				err := s.Boot(context.Background())
				if err != nil {
					t.Fatalf(err.Error())
				}
			}()
		}

		wg.Wait()
	}
}

//
// Below is the helper function to create a test Set.
//

func newTestSet() (*Set, error) {
	var err error

	var newSet *Set
	{
		c := SetConfig{
			Collectors: []Interface{
				&testCollector{},
			},
			Logger: microloggertest.New(),
		}

		newSet, err = NewSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return newSet, nil
}

//
// Below is the test collector implementation used to test the Set
// initialization and Boot.
//

type testCollector struct{}

func (c *testCollector) Collect(ch chan<- prometheus.Metric) error {
	return nil
}

func (c *testCollector) Describe(ch chan<- *prometheus.Desc) error {
	return nil
}
