package blobobject

const (
	prefixMaster = "master"
	prefixWorker = "worker"
)

type ContainerObjectState struct {
	ContainerName      string
	Body               string
	Key                string
	StorageAccountName string
}
