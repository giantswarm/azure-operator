package scalestrategy

type Staircase struct {
}

const (
	safeThreshold = 5
)

func (i Staircase) GetNodeCount(currentCount int64, desiredCount int64) int64 {
	// Cluster size decreased or unchanged.
	if currentCount >= desiredCount {
		return desiredCount
	}

	if desiredCount-currentCount > safeThreshold {
		return currentCount + safeThreshold
	}

	return desiredCount
}
