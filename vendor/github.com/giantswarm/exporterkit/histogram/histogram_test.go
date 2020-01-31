package histogram

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func Example() {
	// Create a new descriptor.
	desc := prometheus.NewDesc(
		"foo_bar_metric",
		"The number of foos that have been bared",
		nil,
		nil,
	)

	// Create a Histogram.
	c := Config{
		BucketLimits: []float64{0, 1, 5},
	}
	h, _ := New(c)

	// Add entries to the Histogram as part of the exporter,
	// e.g: latency of network connections.
	h.Add(0.2)

	// Convert the Histogram into a metric.
	metric := prometheus.MustNewConstHistogram(
		desc, h.Count(), h.Sum(), h.Buckets(),
	)

	fmt.Println(metric)
}

func Test_Add(t *testing.T) {
	testCases := []struct {
		name            string
		config          Config
		addInput        []float64
		expectedCount   uint64
		expectedSum     float64
		expectedBuckets map[float64]uint64
	}{
		{
			name: "case 0: Add 1 input to histogram with one bucket",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput:      []float64{1},
			expectedCount: 1,
			expectedSum:   1,
			expectedBuckets: map[float64]uint64{
				5: 1,
			},
		},

		{
			name: "case 1: Add 1 input between buckets to a histogram with 2 buckets",
			config: Config{
				BucketLimits: []float64{5, 10},
			},
			addInput:      []float64{6},
			expectedCount: 1,
			expectedSum:   6,
			expectedBuckets: map[float64]uint64{
				5:  0,
				10: 1,
			},
		},

		{
			name: "case 2: Add 2 inputs for a histogram with 2 buckets",
			config: Config{
				BucketLimits: []float64{5, 10},
			},
			addInput:      []float64{1, 7},
			expectedCount: 2,
			expectedSum:   8,
			expectedBuckets: map[float64]uint64{
				5:  1,
				10: 2,
			},
		},

		{
			name: "case 3: Add 1 input higher than the bucket",
			config: Config{
				BucketLimits: []float64{10},
			},
			addInput:      []float64{100},
			expectedCount: 1,
			expectedSum:   100,
			expectedBuckets: map[float64]uint64{
				10: 0,
			},
		},

		{
			name: "case 4: Add 2 inputs with negative bucket values",
			config: Config{
				BucketLimits: []float64{-500},
			},
			addInput:      []float64{-501, -499},
			expectedCount: 2,
			expectedSum:   -1000,
			expectedBuckets: map[float64]uint64{
				-500: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := New(tc.config)
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}

			for _, x := range tc.addInput {
				h.Add(x)
			}

			if h.Count() != tc.expectedCount {
				t.Fatalf("count == %v, want %v", h.Count(), tc.expectedCount)
			}
			if h.Sum() != tc.expectedSum {
				t.Fatalf("sum == %v, want %v", h.Sum(), tc.expectedSum)
			}
			if !reflect.DeepEqual(h.Buckets(), tc.expectedBuckets) {
				t.Fatalf("buckets == %v, want %v", h.Buckets(), tc.expectedBuckets)
			}
		})
	}
}

// Test_Concurrency tests that concurrent access to a Histogram is safe.
func Test_Concurrency(t *testing.T) {
	c := Config{
		BucketLimits: []float64{1},
	}
	h, err := New(c)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.Add(1)
		}()
	}

	wg.Wait()
}

// Test_Multi_Write_Iteration_Concurrency tests that concurrent reading and
// iterating of the histogram is safe.
func Test_Multi_Write_Iteration_Concurrency(t *testing.T) {
	c := Config{
		BucketLimits: []float64{1},
	}
	h, err := New(c)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	var writeWaitGroup sync.WaitGroup
	for i := 0; i < 1000; i++ {
		writeWaitGroup.Add(1)

		go func(i int) {
			defer writeWaitGroup.Done()

			h.Add(float64(i))
		}(i)
	}

	var iterateWaitGroup sync.WaitGroup
	for i := 0; i < 1000; i++ {
		iterateWaitGroup.Add(1)

		go func() {
			defer iterateWaitGroup.Done()

			for range h.Buckets() {
			}
		}()
	}

	writeWaitGroup.Wait()
	iterateWaitGroup.Wait()
}
