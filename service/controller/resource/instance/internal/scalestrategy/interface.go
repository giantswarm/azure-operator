package scalestrategy

type Interface interface {
	GetNodeCount(currentCount int64, desiredCount int64) int64
}
