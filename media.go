package anki

import (
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
)

func (c *Collection) GetMedia(name string) (MediaFile, error) {
	path := c.mediaPath(name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &mediaFile{name: name, path: path}, nil
}

func (c *Collection) OpenMedia(name string) (io.ReadCloser, error) {
	return os.Open(c.mediaPath(name))
}

func (c *Collection) AddMedia(name, path string) error {
	return c.CopyMedia(&mediaFile{name: name, path: path})
}

func (c *Collection) WriteMedia(name string, content []byte) error {
	w, err := c.CreateMedia(name)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = w.Write(content)
	return err
}

func (c *Collection) CopyMedia(media MediaFile) error {
	r, err := media.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := c.CreateMedia(media.Name())
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}

func (c *Collection) CreateMedia(name string) (io.WriteCloser, error) {
	// TODO: create media dir
	path := c.mediaPath(name)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &mediaWriteCloser{File: f, path: path}, nil
}

type mediaWriteCloser struct {
	*os.File
	path string
	err  error
}

func (m *mediaWriteCloser) Write(p []byte) (int, error) {
	n, err := m.File.Write(p)
	if err != nil {
		m.err = err
	}
	return n, err
}

func (m *mediaWriteCloser) Close() error {
	err := m.File.Close()
	if err != nil || m.err != nil {
		_ = os.Remove(m.path)
	}
	return err
}

func (c *Collection) DeleteMedia(name string) error {
	return os.Remove(c.mediaPath(name))
}

func (c *Collection) ListMedia() iter.Seq2[MediaFile, error] {
	dir := mediaDir(c.dir)
	return func(yield func(MediaFile, error) bool) {
		fn := func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}

			media := &mediaFile{name: rel, path: path}
			if !yield(media, nil) {
				return filepath.SkipAll
			}

			return nil
		}
		if err := filepath.WalkDir(dir, fn); err != nil {
			yield(nil, err)
		}
	}
}

func (c *Collection) mediaPath(name string) string {
	return filepath.Join(mediaDir(c.dir), name)
}

type MediaFile interface {
	Name() string
	Open() (io.ReadCloser, error)
}

type mediaFile struct {
	name string
	path string
}

func (mf *mediaFile) Name() string {
	return mf.name
}

func (mf *mediaFile) Open() (io.ReadCloser, error) {
	return os.Open(mf.path)
}
