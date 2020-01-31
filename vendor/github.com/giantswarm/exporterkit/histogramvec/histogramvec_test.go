package histogramvec

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func Example() {
	// Create a new descriptor with a label.
	desc := prometheus.NewDesc(
		"fridge_temperature_celsius",
		"The temperature of the fridge in each store",
		[]string{"store_name"},
		nil,
	)

	// Create a HistogramVec.
	c := Config{
		BucketLimits: []float64{0, 1, 2, 3, 4, 5},
	}
	hv, _ := New(c)

	// Define a struct to hold sensor data,
	// here, the temperature of each store's refrigerator.
	type StoreTemperature struct {
		StoreName   string
		Temperature float64
	}

	// Read data from an external system.
	firstSamples := []StoreTemperature{
		{
			StoreName:   "London",
			Temperature: 2.5,
		},
		{
			StoreName:   "Frankfurt",
			Temperature: 3.2,
		},
	}

	// Add the samples to the HistogramVec.
	for _, s := range firstSamples {
		hv.Add(s.StoreName, s.Temperature)
	}

	// Emit each metric, such as in a Prometheus Collector.
	// This will emit Histograms for both the London and Frankfurt stores.
	for label, h := range hv.Histograms() {
		metric := prometheus.MustNewConstHistogram(
			desc,
			h.Count(), h.Sum(), h.Buckets(),
			label,
		)

		fmt.Println(metric)
	}

	// Read data again from an external system.
	// For example, this would be a separate invocation of Collect
	// in a Prometheus Collector.
	secondSamples := []StoreTemperature{
		{
			StoreName:   "London",
			Temperature: 2.6,
		},
		// Note: Frankfurt store has shut down,
		// and does not report fridge temperatures anymore.
	}

	// Add the second set of sample to the HistogramVec.
	for _, s := range secondSamples {
		hv.Add(s.StoreName, s.Temperature)
	}

	// Ensure that any stores that have shut down are removed from the HistogramVec.
	// This should be done on every sampling (ommitted earlier for clarity).
	storeNames := []string{}
	for _, s := range secondSamples {
		storeNames = append(storeNames, s.StoreName)
	}
	hv.Ensure(storeNames)

	// Again, emit each metric. This will only emit a Histogram for the London store,
	// as the Frankfurt store has been removed.
	for label, h := range hv.Histograms() {
		metric := prometheus.MustNewConstHistogram(
			desc,
			h.Count(), h.Sum(), h.Buckets(),
			label,
		)

		fmt.Println(metric)
	}
}

func Test_Add(t *testing.T) {
	type addInput struct {
		label  string
		sample float64
	}
	type histogram struct {
		count   uint64
		sum     float64
		buckets map[float64]uint64
	}

	testCases := []struct {
		name               string
		config             Config
		addInput           []addInput
		expectedHistograms map[string]histogram
	}{
		{
			name: "case 0: Add 1 input to histogramvec",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
			},
			expectedHistograms: map[string]histogram{
				"foo": {
					count: 1,
					sum:   1,
					buckets: map[float64]uint64{
						5: 1,
					},
				},
			},
		},

		{
			name: "case 1: Add 2 differently labelled inputs",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
				{
					label:  "bar",
					sample: 2,
				},
			},
			expectedHistograms: map[string]histogram{
				"foo": {
					count: 1,
					sum:   1,
					buckets: map[float64]uint64{
						5: 1,
					},
				},
				"bar": {
					count: 1,
					sum:   2,
					buckets: map[float64]uint64{
						5: 1,
					},
				},
			},
		},

		{
			name: "case 3: Add multiple inputs to same histogram",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
				{
					label:  "bar",
					sample: 2,
				},
				{
					label:  "foo",
					sample: 2,
				},
			},
			expectedHistograms: map[string]histogram{
				"foo": {
					count: 2,
					sum:   3,
					buckets: map[float64]uint64{
						5: 2,
					},
				},
				"bar": {
					count: 1,
					sum:   2,
					buckets: map[float64]uint64{
						5: 1,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hv, err := New(tc.config)
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}

			for _, x := range tc.addInput {
				if err := hv.Add(x.label, x.sample); err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
			}

			hs := hv.Histograms()
			returnedHistograms := map[string]histogram{}

			for label, h := range hs {
				returnedHistograms[label] = histogram{
					count:   h.Count(),
					sum:     h.Sum(),
					buckets: h.Buckets(),
				}
			}

			if !reflect.DeepEqual(returnedHistograms, tc.expectedHistograms) {
				t.Fatalf("histograms == %v, want %v", returnedHistograms, tc.expectedHistograms)
			}
		})
	}
}

func Test_Ensure(t *testing.T) {
	type addInput struct {
		label  string
		sample float64
	}

	testCases := []struct {
		name           string
		config         Config
		addInput       []addInput
		ensureLabels   []string
		expectedLabels []string
	}{
		{
			name: "case 0: Ensure no labels exist",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
			},
			ensureLabels:   []string{},
			expectedLabels: []string{},
		},

		{
			name: "case 1: Ensure 1 label exists",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
			},
			ensureLabels:   []string{"foo"},
			expectedLabels: []string{"foo"},
		},

		{
			name: "case 2: Ensure 1 label out of 2 exists",
			config: Config{
				BucketLimits: []float64{5},
			},
			addInput: []addInput{
				{
					label:  "foo",
					sample: 1,
				},
				{
					label:  "bar",
					sample: 1,
				},
			},
			ensureLabels:   []string{"foo"},
			expectedLabels: []string{"foo"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hv, err := New(tc.config)
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}

			for _, x := range tc.addInput {
				if err := hv.Add(x.label, x.sample); err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
			}

			hv.Ensure(tc.ensureLabels)

			hs := hv.Histograms()

			returnedLabels := []string{}
			for label, _ := range hs {
				returnedLabels = append(returnedLabels, label)
			}

			if !reflect.DeepEqual(returnedLabels, tc.expectedLabels) {
				t.Fatalf("labels == %v, want %v", returnedLabels, tc.expectedLabels)
			}
		})
	}
}

// Test_Concurrency tests that concurrent access to a HistogramVec is safe.
func Test_Concurrency(t *testing.T) {
	c := Config{
		BucketLimits: []float64{1},
	}
	hv, err := New(c)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hv.Add("foo", 1)
			hv.Ensure([]string{})
		}()
	}

	wg.Wait()
}

// Test_Multi_Write_Iteration_Concurrency tests that concurrent reading and
// iterating of the histograms is safe.
func Test_Multi_Write_Iteration_Concurrency(t *testing.T) {
	c := Config{
		BucketLimits: []float64{1},
	}
	hv, err := New(c)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	var writeWaitGroup sync.WaitGroup
	for i := 0; i < 1000; i++ {
		writeWaitGroup.Add(1)

		go func(i int) {
			defer writeWaitGroup.Done()

			hv.Add(fmt.Sprintf("%v", i), float64(i))
		}(i)
	}

	var iterateWaitGroup sync.WaitGroup
	for i := 0; i < 1000; i++ {
		iterateWaitGroup.Add(1)

		go func() {
			defer iterateWaitGroup.Done()

			for range hv.Histograms() {
			}
		}()
	}

	writeWaitGroup.Wait()
	iterateWaitGroup.Wait()
}
