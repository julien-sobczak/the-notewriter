package helpers

import (
	"crypto/md5"
	"fmt"
	"os"
)

// Hash is an utility to determine a MD5 hash (acceptable as not used for security reasons).
func Hash(bytes []byte) string {
	h := md5.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// HashFromFile reads the file content to determine the hash.
func HashFromFile(path string) (string, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Hash(contentBytes), nil
}

// HashFromFileName reads the file content to determine the hash.
func HashFromFileName(path string) string {
	return Hash([]byte(path))
}
