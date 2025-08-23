package anki

import (
	"archive/zip"
	"database/sql"
	"io"
	"os"
	"time"
)

type Collection struct {
	db        *sql.DB
	dir       string
	isTempDir bool
	props     *props
}

func newCollection(db *sql.DB, dir string, isTempDir bool) (*Collection, error) {
	props, err := loadProps(db)
	if err != nil {
		return nil, err
	}
	return &Collection{
		db:        db,
		dir:       dir,
		isTempDir: isTempDir,
		props:     props,
	}, nil
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

func Create() (*Collection, error) {
	return inTempDir(func(dir string) (*Collection, error) {
		db, err := sqlite3Open(databasePath(dir) + "?_journal=WAL&mode=rwc")
		if err != nil {
			return nil, err
		}

		if err := sqlExecute(db, schemaQuery); err != nil {
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
	return c.props.usn
}

func (c *Collection) ModTime() time.Time {
	return c.props.mod
}

func (c *Collection) SchemdModTime() time.Time {
	return c.props.scm
}

func (c *Collection) LastSyncTime() time.Time {
	return c.props.ls
}

type props struct {
	mod time.Time
	scm time.Time
	ls  time.Time
	usn int64
}

func loadProps(db *sql.DB) (*props, error) {
	const query = `SELECT mod, scm, ls, usn FROM col WHERE id = 1`

	fn := func(_ sqlQueryer, row sqlRow) (*props, error) {
		var mod, scm, ls, usn int64
		if err := row.Scan(&mod, &scm, &ls, &usn); err != nil {
			return nil, err
		}
		return &props{
			mod: time.UnixMilli(mod),
			scm: time.UnixMilli(scm),
			ls:  time.UnixMilli(ls),
			usn: usn,
		}, nil
	}
	return sqlGet(db, fn, query)
}
