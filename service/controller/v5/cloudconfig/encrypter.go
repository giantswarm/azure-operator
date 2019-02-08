package cloudconfig

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"github.com/giantswarm/microerror"
	"io"
)

const (
	blockSize = 32
)

type Encrypter struct {
	key []byte
}

func NewEncrypter() (Encrypter, error) {
	var key []byte

	key = make([]byte, blockSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return Encrypter{}, microerror.Mask(err)
	}

	encrypter := Encrypter{
		key: key,
	}

	return encrypter, nil
}

func (e *Encrypter) EncryptCFBBase64(data []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// generate random initial vector
	encryptedData := make([]byte, aes.BlockSize+len(data))
	iv := encryptedData[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", microerror.Mask(err)
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(encryptedData[aes.BlockSize:], data)

	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

func (e *Encrypter) DecryptCFBBase64(encoded string) ([]byte, error) {
	var block cipher.Block

	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if block, err = aes.NewCipher(e.key); err != nil {
		return nil, microerror.Mask(err)
	}

	iv := encrypted[:aes.BlockSize]
	encrypted = encrypted[aes.BlockSize:]
	decrypted := make([]byte, len(encrypted))

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(decrypted, encrypted)

	return decrypted, nil
}
