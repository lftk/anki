package anki

import (
	"archive/zip"
	"io"

	"github.com/klauspost/compress/zstd"
)

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

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func zipLookup(r *zip.Reader, name string) (*zip.File, bool) {
	for _, f := range r.File {
		if f.Name == name {
			return f, true
		}
	}
	return nil, false
}

func zipReadAll(r *zip.Reader, name string, dcomp bool) ([]byte, error) {
	f, err := zipOpen(r, name, dcomp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

func zipWrite(w *zip.Writer, name string, comp bool, data []byte) error {
	zw, err := zipCreate(w, name, comp)
	if err != nil {
		return err
	}
	defer zw.Close()
	_, err = zw.Write(data)
	return err
}
