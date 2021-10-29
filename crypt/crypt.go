package crypt

import (
	"golang.org/x/crypto/bcrypt"
)

func Hash(password []byte) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func Compare(hashed string, plain []byte) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), plain)
}
