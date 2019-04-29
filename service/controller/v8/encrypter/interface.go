package encrypter

type Interface interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
	GetEncryptionKey() string
	GetInitialVector() string
}
