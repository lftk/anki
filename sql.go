package anki

import (
	"database/sql"
	"iter"
)

// sqlTransact is a helper function to run a database transaction.
func sqlTransact(db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		} else if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// sqlQueryer is an interface for querying the database.
type sqlQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

// sqlRow is an interface for scanning a database row.
type sqlRow interface {
	Scan(dest ...any) error
}

// sqlGet is a generic function to get a single row from the database.
func sqlGet[T any](q sqlQueryer, fn func(sqlQueryer, sqlRow) (T, error), query string, args ...any) (T, error) {
	return fn(q, q.QueryRow(query, args...))
}

// sqlSelect is a generic function to select multiple rows from the database.
func sqlSelect[T any](q sqlQueryer, fn func(sqlQueryer, sqlRow) (T, error), query string, args ...any) ([]T, error) {
	var vals []T
	for val, err := range sqlSelectSeq(q, fn, query, args...) {
		if err != nil {
			return nil, err
		}
		vals = append(vals, val)
	}
	return vals, nil
}

// sqlSelectSeq is a generic function to select multiple rows from the database as a sequence.
func sqlSelectSeq[T any](q sqlQueryer, fn func(sqlQueryer, sqlRow) (T, error), query string, args ...any) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		rows, err := q.Query(query, args...)
		if err != nil {
			var zero T
			yield(zero, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			val, err := fn(q, rows)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			if !yield(val, nil) {
				return
			}
		}
	}
}

// sqlExecer is an interface for executing SQL queries.
type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// sqlInsert is a helper function to execute an INSERT query and return the last insert ID.
func sqlInsert(e sqlExecer, query string, args ...any) (int64, error) {
	r, err := e.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// sqlExecute is a helper function to execute a SQL query.
func sqlExecute(e sqlExecer, query string, args ...any) error {
	_, err := e.Exec(query, args...)
	return err
}

// sqlExt is an interface that combines sqlQueryer and sqlExecer.
type sqlExt interface {
	sqlQueryer
	sqlExecer
}
