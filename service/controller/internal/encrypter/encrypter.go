package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"

	"github.com/giantswarm/microerror"
)

type Config struct {
	Key []byte
	IV  []byte
}

type Encrypter struct {
	key []byte
	iv  []byte
}

func New(config Config) (*Encrypter, error) {
	if config.Key == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Key must not be empty", config)
	}
	if config.IV == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.IV must not be empty", config)
	}

	encrypter := &Encrypter{
		key: config.Key,
		iv:  config.IV,
	}

	return encrypter, nil
}

func (e *Encrypter) Encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	encryptedData := make([]byte, len(data))

	cfb := cipher.NewCFBEncrypter(block, e.iv)
	cfb.XORKeyStream(encryptedData, data)

	return encryptedData, nil
}

func (e *Encrypter) Decrypt(encrypted []byte) ([]byte, error) {
	var block cipher.Block
	var err error

	if block, err = aes.NewCipher(e.key); err != nil {
		return nil, microerror.Mask(err)
	}

	decrypted := make([]byte, len(encrypted))

	cfb := cipher.NewCFBDecrypter(block, e.iv)
	cfb.XORKeyStream(decrypted, encrypted)

	return decrypted, nil
}

// GetEncryptionKey returns hex of the key, which is used for certificates encryption.
func (e *Encrypter) GetEncryptionKey() string {
	return hex.EncodeToString(e.key)
}

// GetInitialVector returns hex of the initial vector, which is used in certificate encryption.
func (e *Encrypter) GetInitialVector() string {
	return hex.EncodeToString(e.iv)
}
