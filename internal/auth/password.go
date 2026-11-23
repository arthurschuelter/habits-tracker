package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	// RFC 9106's option for memory constrained environments.
	memoryKiB        = 64 * 1024
	passes           = 3
	lanes            = 4
	saltSize         = 16
	keySize          = 32
	hashPrefix       = "$argon2id$v=19$m=65536,t=3,p=4$"
	maxPasswordBytes = 1024
)

var errPasswordRule = &ValidationError{Field: "password", Message: "must contain at least 8 characters and at most 1024 bytes"}

// ValidatePassword returns a *ValidationError for passwords that break the length rules.
func ValidatePassword(password []byte) error {
	if !utf8.Valid(password) || len(password) > maxPasswordBytes ||
		utf8.RuneCount(password) < 8 || strings.TrimSpace(string(password)) == "" {
		return errPasswordRule
	}
	return nil
}

// HashPassword returns a *ValidationError for passwords that break the length rules.
func HashPassword(password []byte) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	var salt [saltSize]byte
	_, _ = rand.Read(salt[:])
	key := argon2.IDKey(password, salt[:], passes, memoryKiB, lanes, keySize)
	return hashPrefix + base64.RawStdEncoding.EncodeToString(salt[:]) + "$" +
		base64.RawStdEncoding.EncodeToString(key), nil
}

func verifyPassword(password string, encoded string) bool {
	// Accept only the format we issue. A malformed stored value cannot select
	// arbitrary memory or CPU costs. Unknown users still perform the same work.
	body, prefixOK := strings.CutPrefix(encoded, hashPrefix)
	saltText, keyText, separatorOK := strings.Cut(body, "$")
	salt, saltErr := base64.RawStdEncoding.DecodeString(saltText)
	key, keyErr := base64.RawStdEncoding.DecodeString(keyText)
	valid := prefixOK && separatorOK && saltErr == nil && keyErr == nil &&
		len(salt) == saltSize && len(key) == keySize
	if !valid {
		salt = make([]byte, saltSize)
		key = make([]byte, keySize)
	}
	candidate := argon2.IDKey([]byte(password), salt, passes, memoryKiB, lanes, keySize)
	return subtle.ConstantTimeCompare(candidate, key) == 1 && valid
}
