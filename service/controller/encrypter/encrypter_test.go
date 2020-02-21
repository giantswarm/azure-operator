package encrypter

import (
	"testing"
)

func Test_EncryptCFB(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
	}{
		{
			Name:  "plain text encryption",
			Input: []byte("testtext"),
		},
	}

	testKey := []byte("12345678901234567890123456789012")
	testIV := []byte("1234567891234567")
	c := Config{
		Key: testKey,
		IV:  testIV,
	}

	encrypter, err := New(c)
	if err != nil {
		t.Errorf("failed to create encrypter, %v", err)
	}

	for i, tc := range testCases {
		encrypted, err := encrypter.Encrypt(tc.Input)
		if err != nil {
			t.Errorf("case %d: %s: expected err = nil, got %v", i, tc.Name, err)
		}

		decrypted, err := encrypter.Decrypt(encrypted)
		if err != nil {
			t.Errorf("case %d: %s: expected err = nil, got %v", i, tc.Name, err)
		}

		if string(tc.Input) != string(decrypted) {
			t.Errorf("case %d: %s: expected %s, got %s", i, tc.Name, tc.Input, decrypted)
		}
	}
}
