package scalestrategy

type Quick struct {
}

func (i Quick) GetNodeCount(currentCount int64, desiredCount int64) int64 {
	return desiredCount
}
