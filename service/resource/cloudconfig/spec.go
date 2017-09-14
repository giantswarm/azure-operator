package cloudconfig

type CloudConfigBlob struct {
	Name               string
	StorageAccountName string
	ContainerName      string
	BlobName           string
	Data               string
}
