package cloudconfig

import (
	"testing"
)

const (
	TestKey = "12345678901234567890123456789012"
)

func Test_EncryptCFBBase64(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
	}{
		{
			Name:  "plain text encryption",
			Input: []byte("testtext"),
		},
	}

	encrypter, err := NewEncrypter()
	if err != nil {
		t.Errorf("failed to create encrypter, %v", err)
	}

	for i, tc := range testCases {
		encrypted, err := encrypter.EncryptCFBBase64(tc.Input)
		if err != nil {
			t.Errorf("case %d: %s: expected err = nil, got %v", i, tc.Name, err)
		}

		decrypted, err := encrypter.DecryptCFBBase64(encrypted)
		if err != nil {
			t.Errorf("case %d: %s: expected err = nil, got %v", i, tc.Name, err)
		}

		if string(tc.Input) != string(decrypted) {
			t.Errorf("case %d: %s: expected %s, got %s", i, tc.Name, tc.Input, decrypted)
		}
	}
}
