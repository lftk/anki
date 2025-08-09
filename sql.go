package anki

import (
	"database/sql"
	"iter"

	"github.com/google/uuid"
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

func sqlQuery[T any](q sqlQueryer, fn func(sqlQueryer, sqlRow) (T, error), query string, args ...any) (T, error) {
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
			item, err := fn(q, rows)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}

			if !yield(item, nil) {
				return
			}
		}
	}
}

func generateGUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
