package ipam

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	capzExpV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/pkg/locker"
)

func Test_SubnetAllocator(t *testing.T) {
	testCases := []struct {
		name string

		checker            Checker
		collector          Collector
		networkRangeGetter NetworkRangeGetter
		persister          Persister
	}{
		{
			name: "case 0 allocate first subnet",

			checker:            NewTestChecker(true),
			collector:          NewTestCollector([]net.IPNet{}),
			networkRangeGetter: NewTestNetworkRangeGetter(mustParseCIDR("10.100.0.0/16"), 24),
			persister:          NewTestPersister(mustParseCIDR("10.100.0.0/24")),
		},
		{
			name: "case 1 allocate fourth subnet",

			checker: NewTestChecker(true),
			collector: NewTestCollector([]net.IPNet{
				mustParseCIDR("10.100.0.0/24"),
				mustParseCIDR("10.100.1.0/24"),
				mustParseCIDR("10.100.3.0/24"),
			}),
			networkRangeGetter: NewTestNetworkRangeGetter(mustParseCIDR("10.100.0.0/16"), 24),
			persister:          NewTestPersister(mustParseCIDR("10.100.2.0/24")),
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var err error

			var mutexLocker locker.Interface
			{
				c := locker.MutexLockerConfig{
					Logger: microloggertest.New(),
				}

				mutexLocker, err = locker.NewMutexLocker(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			var newResource *Resource
			{
				c := Config{
					Checker:            tc.checker,
					Collector:          tc.collector,
					Locker:             mutexLocker,
					Logger:             microloggertest.New(),
					NetworkRangeGetter: tc.networkRangeGetter,
					Persister:          tc.persister,
				}

				newResource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			err = newResource.EnsureCreated(context.Background(), &capzExpV1alpha3.AzureMachinePool{})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mustParseCIDR(val string) net.IPNet {
	_, n, err := net.ParseCIDR(val)
	if err != nil {
		panic(err)
	}

	return *n
}
