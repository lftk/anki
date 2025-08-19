package anki

import (
	"archive/zip"
	"database/sql"
	_ "embed"
	"io"
	"os"
	"time"
)

type Collection struct {
	db        *sql.DB
	dir       string
	isTempDir bool

	mod time.Time
	scm time.Time
	usn int64
	ls  time.Time
}

func newCollection(db *sql.DB, dir string, isTempDir bool) (*Collection, error) {
	col := &Collection{
		db:        db,
		dir:       dir,
		isTempDir: isTempDir,
	}
	if err := col.load(); err != nil {
		_ = col.Close()
		return nil, err
	}
	if err := os.MkdirAll(col.mediaDir(), 0755); err != nil {
		_ = col.Close()
		return nil, err
	}
	return col, nil
}

func (c *Collection) load() error {
	const query = `SELECT mod, scm, usn, ls FROM col WHERE id = 1`

	var modMilli, scmSec, lsSec int64
	err := c.db.QueryRow(query).Scan(&modMilli, &scmSec, &c.usn, &lsSec)
	if err != nil {
		return err
	}

	c.mod = time.UnixMilli(modMilli)
	c.scm = time.UnixMilli(scmSec)
	c.ls = time.Unix(lsSec, 0)
	return nil
}

func inTempDir(fn func(dir string) (*Collection, error)) (*Collection, error) {
	dir, err := os.MkdirTemp("", "anki-*")
	if err != nil {
		return nil, err
	}
	col, err := fn(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return col, nil
}

//go:embed schema.sql
var ddl string

func Create() (*Collection, error) {
	return inTempDir(func(dir string) (*Collection, error) {
		db, err := sqlite3Open(databasePath(dir) + "?_journal=WAL&mode=rwc")
		if err != nil {
			return nil, err
		}

		if err := sqlExecute(db, ddl); err != nil {
			_ = db.Close()
			return nil, err
		}

		return newCollection(db, dir, true)
	})
}

func Open(col string) (*Collection, error) {
	return inTempDir(func(dir string) (*Collection, error) {
		r, err := zip.OpenReader(col)
		if err != nil {
			return nil, err
		}
		defer r.Close()

		if err = Unpack(&r.Reader, dir); err != nil {
			return nil, err
		}

		return loadDir(dir, true)
	})
}

func ReadFrom(r io.ReaderAt, size int64) (*Collection, error) {
	return inTempDir(func(dir string) (*Collection, error) {
		zr, err := zip.NewReader(r, size)
		if err != nil {
			return nil, err
		}

		if err = Unpack(zr, dir); err != nil {
			return nil, err
		}

		return loadDir(dir, true)
	})
}

func LoadDir(dir string) (*Collection, error) {
	return loadDir(dir, false)
}

func loadDir(dir string, isTempDir bool) (*Collection, error) {
	db, err := sqlite3Open(databasePath(dir) + "?_journal=WAL")
	if err != nil {
		return nil, err
	}
	return newCollection(db, dir, isTempDir)
}

func (c *Collection) WriteTo(w io.Writer) (int64, error) {
	if err := c.flush(); err != nil {
		return 0, err
	}
	sw := &statsWriter{w: w}
	zw := zip.NewWriter(sw)
	if err := Pack(zw, c.dir); err != nil {
		return 0, err
	}
	return sw.n, zw.Close()
}

type statsWriter struct {
	n int64
	w io.Writer
}

func (sw *statsWriter) Write(p []byte) (int, error) {
	n, err := sw.w.Write(p)
	sw.n += int64(n)
	return n, err
}

func (c *Collection) SaveAs(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = c.WriteTo(f)
	return err
}

func (c *Collection) DumpTo(dir string) error {
	if err := c.flush(); err != nil {
		return err
	}
	return backup(c.dir, dir)
}

func (c *Collection) Close() error {
	defer func() {
		if c.isTempDir {
			_ = os.RemoveAll(c.dir)
		}
	}()
	return c.db.Close()
}

func (c *Collection) flush() error {
	return sqlExecute(c.db, "PRAGMA wal_checkpoint(FULL)")
}

func (c *Collection) USN() int64 {
	return c.usn
}

func (c *Collection) ModTime() time.Time {
	return c.mod
}

func (c *Collection) SchemdModTime() time.Time {
	return c.scm
}

func (c *Collection) LastSyncTime() time.Time {
	return c.ls
}
