package crmapper

import (
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/giantswarm/microerror"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

func getAvailabilityZones(azureCluster capzv1alpha3.AzureCluster) ([]int, error) {
	azs := []int{}

	for _, az := range azureCluster.Status.FailureDomains.GetIDs() {
		if az == nil {
			return nil, microerror.Maskf(executionFailedError, "nil in AzureCluster %q FailureDomains", azureCluster.Name)
		}

		n, err := strconv.ParseInt(*az, 10, 64)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azs = append(azs, int(n))
	}

	return azs, nil
}

func getRandomAZs(count int) []int {
	azs := generateRandomIntSlice(count)
	sort.Ints(sort.IntSlice(azs))

	return azs
}

func generateRandomIntSlice(length int) []int {
	rand.Seed(time.Now().UnixNano())
	randomIntSlice := make([]int, length)

	existingNumbers := map[int]bool{}
	i := 0

	for i < length {
		randomNumber := generateRandomNumber(1, length)

		if !existingNumbers[randomNumber] {
			randomIntSlice[i] = randomNumber
			existingNumbers[randomNumber] = true

			i++
		}
	}

	return randomIntSlice
}

func generateRandomNumber(min, max int) int {
	return min + rand.Intn(max-min+1)
}
