package anki

import (
	"archive/zip"
	"io"

	"github.com/klauspost/compress/zstd"
)

// zipOpen opens a file from a zip archive, with optional zstd decompression.
func zipOpen(r *zip.Reader, name string, dcomp bool) (io.ReadCloser, error) {
	f, err := r.Open(name)
	if err != nil {
		return nil, err
	}
	if dcomp {
		dec, err := zstd.NewReader(f)
		if err != nil {
			return nil, err
		}
		return dec.IOReadCloser(), nil
	}
	return f, nil
}

// zipCreate creates a file in a zip archive, with optional zstd compression.
func zipCreate(w *zip.Writer, name string, comp bool) (io.WriteCloser, error) {
	zw, err := w.Create(name)
	if err != nil {
		return nil, err
	}
	if comp {
		return zstd.NewWriter(zw)
	}
	return nopWriteCloser{zw}, nil
}

// nopWriteCloser is a no-op WriteCloser.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// zipLookup looks up a file in a zip archive.
func zipLookup(r *zip.Reader, name string) (*zip.File, bool) {
	for _, f := range r.File {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

// zipReadAll reads all content from a file in a zip archive.
func zipReadAll(r *zip.Reader, name string, dcomp bool) ([]byte, error) {
	f, err := zipOpen(r, name, dcomp)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck

	return io.ReadAll(f)
}

// zipWrite writes data to a file in a zip archive.
func zipWrite(w *zip.Writer, name string, comp bool, data []byte) error {
	zw, err := zipCreate(w, name, comp)
	if err != nil {
		return err
	}
	defer zw.Close() //nolint:errcheck

	_, err = zw.Write(data)
	return err
}
