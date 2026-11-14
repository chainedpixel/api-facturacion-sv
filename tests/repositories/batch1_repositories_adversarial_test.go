package repositories

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/adapters/repositories"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
)

type counterState struct {
	mu      sync.Mutex
	rowLock sync.Mutex
	exists  bool
	last    int
	updates atomic.Int64
	inserts atomic.Int64
}

type counterConnector struct{ state *counterState }

func (c *counterConnector) Connect(context.Context) (driver.Conn, error) {
	return &counterConn{state: c.state}, nil
}
func (c *counterConnector) Driver() driver.Driver { return counterDriver{} }

type counterDriver struct{}

func (counterDriver) Open(string) (driver.Conn, error) { return nil, errors.New("unsupported") }

type counterConn struct {
	state    *counterState
	holdsRow bool
}

func (c *counterConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unsupported") }
func (c *counterConn) Close() error                        { c.release(); return nil }
func (c *counterConn) Begin() (driver.Tx, error)           { return &counterTx{conn: c}, nil }
func (c *counterConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &counterTx{conn: c}, nil
}

func (c *counterConn) release() {
	if c.holdsRow {
		c.holdsRow = false
		c.state.rowLock.Unlock()
	}
}

type counterTx struct{ conn *counterConn }

func (t *counterTx) Commit() error   { t.conn.release(); return nil }
func (t *counterTx) Rollback() error { t.conn.release(); return nil }

var columnsOfUpdate = regexp.MustCompile("SET (.*) WHERE")
var columnsOfInsert = regexp.MustCompile(`\(([^)]*)\) VALUES`)

func positionOf(columns, name string) int {
	for index, column := range strings.Split(columns, ",") {
		if strings.Contains(column, "`"+name+"`") {
			return index
		}
	}
	return -1
}

// QueryContext emulates a locked read: it only returns the counter row when the connection holds the row lock,
// so a locked read that started before the row existed sees no row, as InnoDB does, and the later insert fails
// with a duplicate key.
func (c *counterConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "FOR UPDATE") {
		c.state.mu.Lock()
		exists := c.state.exists
		c.state.mu.Unlock()
		if exists {
			c.state.rowLock.Lock()
			c.holdsRow = true
		}
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	columns := []string{"id", "branch_id", "dte_type", "year", "last_number", "created_at", "updated_at"}
	rows := &counterRows{columns: columns}
	lockedReadWithoutRow := strings.Contains(query, "FOR UPDATE") && !c.holdsRow
	if c.state.exists && !lockedReadWithoutRow {
		rows.data = [][]driver.Value{{int64(1), int64(1), "01", int64(2026), int64(c.state.last), time.Now(), time.Now()}}
	}
	return rows, nil
}

func (c *counterConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch {
	case strings.HasPrefix(query, "UPDATE"):
		position := positionOf(columnsOfUpdate.FindStringSubmatch(query)[1], "last_number")
		c.state.last = int(args[position].Value.(int64))
		c.state.updates.Add(1)
	case strings.HasPrefix(query, "INSERT"):
		if c.state.exists {
			return nil, errors.New("Error 1062: Duplicate entry for key idx_branch_dte_year")
		}
		position := positionOf(columnsOfInsert.FindStringSubmatch(query)[1], "last_number")
		c.state.last = int(args[position].Value.(int64))
		c.state.exists = true
		c.state.inserts.Add(1)
	}
	return sqlmockResult{}, nil
}

type counterRows struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func (r *counterRows) Columns() []string { return r.columns }
func (r *counterRows) Close() error      { return nil }
func (r *counterRows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.index])
	r.index++
	return nil
}

func newCounterGorm(t *testing.T, state *counterState) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(&counterConnector{state: state})
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB
}

// TestAdversarialControlNumberConcurrentGeneration drives GetNext from many goroutines against a stateful fake driver
// that honours SELECT ... FOR UPDATE row locks, proving every caller receives a distinct consecutive number. Real
// InnoDB locking is not exercised here.
func TestAdversarialControlNumberConcurrentGeneration(t *testing.T) {
	test.TestMain(t)

	for _, preExisting := range []bool{false, true} {
		name := "first number race"
		if preExisting {
			name = "existing counter"
		}
		t.Run(name, func(t *testing.T) {
			state := &counterState{exists: preExisting}
			if preExisting {
				state.last = 100
			}
			repo := repositories.NewControlNumberRepository(newCounterGorm(t, state))

			const workers = 80
			numbers := make(chan int, workers)
			errs := make(chan error, workers)
			start := make(chan struct{})
			var group sync.WaitGroup
			for index := 0; index < workers; index++ {
				group.Add(1)
				go func() {
					defer group.Done()
					<-start
					next, err := repo.GetNext(context.Background(), "01", 1)
					if err != nil {
						errs <- err
						return
					}
					numbers <- next
				}()
			}
			close(start)
			group.Wait()
			close(numbers)
			close(errs)

			for err := range errs {
				t.Errorf("GetNext failed under concurrency: %v", err)
			}

			var got []int
			for number := range numbers {
				got = append(got, number)
			}
			sort.Ints(got)
			require.Len(t, got, workers)

			base := 1
			if preExisting {
				base = 101
			}
			for index, number := range got {
				assert.Equal(t, base+index, number, "numbers must be distinct and consecutive")
			}
			assert.Equal(t, base+workers-1, state.last)
			if !preExisting {
				assert.EqualValues(t, 1, state.inserts.Load(), "the counter row must be inserted exactly once")
			}
		})
	}
}

// TestAdversarialControlNumberFailurePaths covers the retry limit, a cancelled context and the first insert value.
func TestAdversarialControlNumberFailurePaths(t *testing.T) {
	test.TestMain(t)
	columns := []string{"id", "branch_id", "dte_type", "last_number", "year"}

	t.Run("both attempts fail returns the error after exactly two attempts", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		boom := errors.New("lock wait timeout exceeded")
		for attempt := 0; attempt < 2; attempt++ {
			mock.ExpectBegin()
			mock.ExpectQuery("FOR UPDATE").WillReturnError(boom)
			mock.ExpectRollback()
		}

		next, err := repositories.NewControlNumberRepository(db).GetNext(context.Background(), "01", 1)

		require.Error(t, err)
		assert.ErrorIs(t, err, boom)
		assert.Zero(t, next)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("first insert persists last_number 1 and returns 1", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("FOR UPDATE").WillReturnRows(sqlmock.NewRows(columns))
		mock.ExpectExec("INSERT INTO `control_number_sequences`").
			WithArgs(uint(7), "03", sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		next, err := repositories.NewControlNumberRepository(db).GetNext(context.Background(), "03", 7)

		require.NoError(t, err)
		assert.Equal(t, 1, next)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cancelled context fails without a number", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectBegin().WillReturnError(context.Canceled)
		mock.ExpectBegin().WillReturnError(context.Canceled)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		next, err := repositories.NewControlNumberRepository(db).GetNext(ctx, "01", 1)

		require.Error(t, err)
		assert.Zero(t, next)
	})

	t.Run("update failure rolls back and retries once", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("FOR UPDATE").WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 1, "01", 9, 2026))
		mock.ExpectExec("UPDATE `control_number_sequences`").WillReturnError(errors.New("deadlock"))
		mock.ExpectRollback()
		mock.ExpectBegin()
		mock.ExpectQuery("FOR UPDATE").WillReturnRows(sqlmock.NewRows(columns).AddRow(1, 1, "01", 9, 2026))
		mock.ExpectExec("UPDATE `control_number_sequences`").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		next, err := repositories.NewControlNumberRepository(db).GetNext(context.Background(), "01", 1)

		require.NoError(t, err)
		assert.Equal(t, 10, next)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestAdversarialCleanerQueryIsBranchScoped verifies the cleaner exclusion only considers persisted documents of the same branch.
func TestAdversarialCleanerQueryIsBranchScoped(t *testing.T) {
	test.TestMain(t)

	gormDB, mock := newReservedSequenceMockDB(t)
	repo := repositories.NewReservedSequenceRepository(gormDB)

	mock.ExpectQuery(`(?s)is_contingency = \?.*expires_at < \?.*NOT EXISTS \(\s*SELECT 1 FROM dte_details d\s+JOIN dte_documents dd ON dd.document_id = d.id\s+WHERE dd.branch_id = reserved_sequence_numbers.branch_id\s+AND d.dte_type = reserved_sequence_numbers.dte_type`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "branch_id", "dte_type", "sequence_number", "year", "status"}))

	_, err := repo.GetExpiredNonContingencyReservations(context.Background())

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAdversarialIssuerNITFilterIsParameterized verifies hostile NIT values are bound as arguments and never reach the SQL text.
func TestAdversarialIssuerNITFilterIsParameterized(t *testing.T) {
	test.TestMain(t)

	hostile := []string{
		"0614-010101-000-0",
		"' OR '1'='1",
		"1'; DROP TABLE dte_documents; --",
		"\" OR 1=1 --",
		"0614' UNION SELECT password FROM users WHERE '1'='1",
		"%",
		"\\",
		"0614-010101-000-0\x00",
		strings.Repeat("9", 5000),
	}

	for _, nit := range hostile {
		t.Run(nit[:min(len(nit), 20)], func(t *testing.T) {
			db, mock := newDTERepoMockDB(t)
			mock.ExpectQuery(`branch_id IN \(SELECT branch_offices.id FROM .branch_offices. JOIN users ON users.id = branch_offices.user_id WHERE users.nit = \?\)`).
				WithArgs(nit).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			_, err := repositories.NewDTERepository(db).GetTotalCount(dteRepoCtx(), &dte.DTEFilters{IssuerNIT: nit, BranchID: 99, IncludeAll: true})

			require.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestAdversarialFilterPrecedence verifies the NIT wins over the branch, an empty NIT keeps the branch filter and no filter adds none.
func TestAdversarialFilterPrecedence(t *testing.T) {
	test.TestMain(t)

	t.Run("empty NIT falls back to the branch filter", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectQuery(`dte_documents.branch_id = \?`).WithArgs(uint(5)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		_, err := repositories.NewDTERepository(db).GetTotalCount(dteRepoCtx(), &dte.DTEFilters{BranchID: 5, IncludeAll: false})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NIT replaces the branch filter entirely", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectQuery(`^SELECT count\(\*\) FROM .dte_documents. JOIN dte_details ON dte_documents.document_id = dte_details.id WHERE dte_documents.branch_id IN \(SELECT branch_offices.id FROM .branch_offices. JOIN users ON users.id = branch_offices.user_id WHERE users.nit = \?\)$`).
			WithArgs("NIT-A").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		_, err := repositories.NewDTERepository(db).GetTotalCount(dteRepoCtx(), &dte.DTEFilters{BranchID: 5, IssuerNIT: "NIT-A"})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no branch and no NIT adds no ownership filter", func(t *testing.T) {
		db, mock := newDTERepoMockDB(t)
		mock.ExpectQuery(`^SELECT count\(\*\) FROM .dte_documents. JOIN dte_details ON dte_documents.document_id = dte_details.id$`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		_, err := repositories.NewDTERepository(db).GetTotalCount(dteRepoCtx(), &dte.DTEFilters{})

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

type sqlmockResult struct{}

func (sqlmockResult) LastInsertId() (int64, error) { return 1, nil }
func (sqlmockResult) RowsAffected() (int64, error) { return 1, nil }
