package crypt

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash hashing given password
func Hash(password []byte) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Compare compares given plain password with hashed password
func Compare(hashed string, plain []byte) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), plain)
}
