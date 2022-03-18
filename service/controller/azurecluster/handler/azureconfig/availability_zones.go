package azureconfig

import (
	"sort"
	"strconv"

	"github.com/giantswarm/microerror"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
)

func getAvailabilityZones(masters, workers []capz.AzureMachine) ([]int, error) {
	azs := []int{}

	for _, m := range append(masters, workers...) {
		if m.Spec.FailureDomain == nil || *m.Spec.FailureDomain == "" {
			continue
		}

		n, err := strconv.ParseInt(*m.Spec.FailureDomain, 10, 64)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azs = append(azs, int(n))
	}

	azs = sortAndUniq(azs)

	return azs, nil
}

func sortAndUniq(xs []int) []int {
	if xs == nil {
		return []int{}
	}

	sort.Ints(xs)

	for i := 0; i < len(xs)-1; i++ {
		if xs[i] == xs[i+1] {
			xs = append(xs[:i], xs[i+1:]...)
			i--
		}
	}

	return xs
}
