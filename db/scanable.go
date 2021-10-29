package db

type Scanable interface {
	Scan(dest ...interface{}) error
}
