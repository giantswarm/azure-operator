package scalestrategy

type Incremental struct {
}

func (i Incremental) GetNodeCount(currentCount int64, desiredCount int64) int64 {
	if currentCount < desiredCount {
		return currentCount + 1
	}

	if currentCount > desiredCount {
		return currentCount - 1
	}

	return currentCount
}
