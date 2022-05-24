package db

import (
	"context"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DBTestSuite struct {
	suite.Suite
	db   *DB
	mock sqlmock.Sqlmock
}

// SetupTest will be run before every test in the suite.
func (s *DBTestSuite) SetupTest() {
	mockdb, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
	)
	if err != nil {
		panic(err)
	}

	mock.MatchExpectationsInOrder(true)
	s.db = &DB{
		DB: mockdb,
	}
	s.mock = mock
}

// TestTransactionCommit will test the transaction ended with commit.
func (s *DBTestSuite) TestTransactionCommit() {
	ctx := context.Background()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectCommit()

	// test
	err := s.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, err := s.db.Exec(txCtx, q, "name", "email")
		return err
	})

	assert.NoError(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

// TestTransactionRollback will test the transaction ended with rollback.
func (s *DBTestSuite) TestTransactionRollback() {
	ctx := context.Background()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectRollback()

	// test
	err := s.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, err := s.db.Exec(txCtx, q, "name", "email")
		assert.NoError(s.T(), err)

		return fmt.Errorf("application error occurred")
	})

	assert.Error(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

// TestTransactionRollbackWithPanic will test the transaction ended with rollback.
func (s *DBTestSuite) TestTransactionRollbackWithPanic() {
	ctx := context.Background()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectRollback()

	// test
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()

		err = s.db.WithTransaction(ctx, func(txCtx context.Context) error {
			_, txErr := s.db.Exec(txCtx, q, "name", "email")
			assert.NoError(s.T(), txErr)

			panic("application error occurred")
		})
	}()

	assert.Error(s.T(), err)
	assert.Equal(s.T(), "application error occurred", err.Error())
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func TestDBTestSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
