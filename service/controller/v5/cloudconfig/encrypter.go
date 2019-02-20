package cloudconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/giantswarm/microerror"
)

const (
	keySize = 32
)

type Encrypter struct {
	key []byte
	iv  []byte
}

func NewEncrypter() (Encrypter, error) {
	var key, iv []byte

	key = make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return Encrypter{}, microerror.Mask(err)
	}

	iv = make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return Encrypter{}, microerror.Mask(err)
	}

	encrypter := Encrypter{
		key: key,
		iv:  iv,
	}

	return encrypter, nil
}

func (e *Encrypter) EncryptCFB(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	encryptedData := make([]byte, len(data))

	cfb := cipher.NewCFBEncrypter(block, e.iv)
	cfb.XORKeyStream(encryptedData, data)

	return encryptedData, nil
}

func (e *Encrypter) DecryptCFB(encrypted []byte) ([]byte, error) {
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
