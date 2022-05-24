package db

type contextKey struct {
	name string
}

var txKey = contextKey{name: "tx"}
