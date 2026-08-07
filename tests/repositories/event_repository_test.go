package repositories

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func newEventRepoMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm: %v", err)
	}
	return gdb, mock
}

func TestEventRepository_Save(t *testing.T) {
	test.TestMain(t)

	gdb, mock := newEventRepoMock(t)
	repo := repositories.NewEventRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `domain_events`")).
		WithArgs(uint64(7), uint64(3), "contingency.activated", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := repo.Save(context.Background(), "contingency.activated", 7, 3, `{"x":1}`, time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
