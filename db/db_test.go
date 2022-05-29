package db

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func (s *DBTestSuite) TearDownTest() {
	s.db.Close()
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
	targetErr := fmt.Errorf("application error occurred")
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()

		err = s.db.WithTransaction(ctx, func(txCtx context.Context) error {

			_, txErr := s.db.Exec(txCtx, q, "name", "email")
			assert.NoError(s.T(), txErr)

			panic(targetErr)
		})
	}()

	assert.Error(s.T(), err)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *DBTestSuite) TestTransactionRollbackParentCancel() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).
		WillDelayFor(50 * time.Millisecond).
		WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectRollback()

	// test
	err := s.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, err := s.db.Exec(txCtx, q, "name", "email")
		time.Sleep(51 * time.Millisecond)
		return err
	})

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, context.DeadlineExceeded)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *DBTestSuite) TestTransactionRollbackParentCancelAA() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).
		WillDelayFor(50 * time.Millisecond).
		WillReturnError(fmt.Errorf("dummy error"))
	s.mock.ExpectRollback()

	// test
	err := s.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, err := s.db.Exec(txCtx, q, "name", "email")
		time.Sleep(51 * time.Millisecond)
		return err
	})

	assert.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, context.DeadlineExceeded)
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func (s *DBTestSuite) TestTransactionRollbackParentCancelBBPanic() {
	ctx := context.Background()

	// mock
	q := "INSERT INTO `users` (`name`, `email`) VALUES (?, ?)"
	s.mock.ExpectBegin()
	s.mock.ExpectExec(q).WillReturnResult(sqlmock.NewResult(1, 1))
	s.mock.ExpectRollback()

	// test
	targetErr := fmt.Errorf("application error occurred")
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()

		err = s.db.WithTransaction(ctx, func(txCtx context.Context) error {
			if _, eerr := s.db.Exec(txCtx, q, "name", "email"); eerr != nil {
				panic(eerr)
			}
			panic(targetErr)
		})
	}()

	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), targetErr.Error())
	assert.NoError(s.T(), s.mock.ExpectationsWereMet())
}

func TestDBTestSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
