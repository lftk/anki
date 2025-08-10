package anki

import (
	"database/sql"
	"iter"
)

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

type sqlQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

type sqlRow interface {
	Scan(dest ...any) error
}

func sqlGet[T any](q sqlQueryer, fn func(sqlQueryer, sqlRow) (T, error), query string, args ...any) (T, error) {
	return fn(q, q.QueryRow(query, args...))
}

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

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func sqlInsert(e sqlExecer, query string, args ...any) (int64, error) {
	r, err := e.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}
