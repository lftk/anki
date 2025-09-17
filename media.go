package anki

import (
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
)

// GetMedia gets a media file by name.
func (c *Collection) GetMedia(name string) (Media, error) {
	path := c.mediaPath(name)
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return &media{name: name, path: path}, nil
}

// OpenMedia opens a media file for reading.
func (c *Collection) OpenMedia(name string) (io.ReadCloser, error) {
	m, err := c.GetMedia(name)
	if err != nil {
		return nil, err
	}
	return m.Open()
}

// AddMedia adds a media file from a path.
func (c *Collection) AddMedia(name, path string) error {
	return c.CopyMedia(&media{name: name, path: path})
}

// WriteMedia writes content to a media file.
func (c *Collection) WriteMedia(name string, content []byte) error {
	w, err := c.CreateMedia(name)
	if err != nil {
		return err
	}
	defer w.Close() //nolint:errcheck

	_, err = w.Write(content)
	return err
}

// CopyMedia copies a media file to the collection.
func (c *Collection) CopyMedia(media Media) error {
	r, err := media.Open()
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	w, err := c.CreateMedia(media.Name())
	if err != nil {
		return err
	}
	defer w.Close() //nolint:errcheck

	_, err = io.Copy(w, r)
	return err
}

// CreateMedia creates a new media file and returns a writer.
func (c *Collection) CreateMedia(name string) (io.WriteCloser, error) {
	path := c.mediaPath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &mediaWriteCloser{File: f, path: path}, nil
}

// mediaWriteCloser is a writer that removes the file if an error occurs.
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

// DeleteMedia deletes a media file.
func (c *Collection) DeleteMedia(name string) error {
	return os.Remove(c.mediaPath(name))
}

// ListMediaOptions specifies options for listing media files.
type ListMediaOptions struct {
	// A glob pattern to filter media files by name.
	Pattern *string
}

// ListMedia lists all media files.
func (c *Collection) ListMedia(opts *ListMediaOptions) iter.Seq2[Media, error] {
	dir := c.mediaDir()
	return func(yield func(Media, error) bool) {
		fn := func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}

			name, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}

			if opts != nil && opts.Pattern != nil {
				matched, err := filepath.Match(*opts.Pattern, name)
				if err != nil {
					return err
				}
				if !matched {
					return nil
				}
			}

			if !yield(&media{name: name, path: path}, nil) {
				return filepath.SkipAll
			}
			return nil
		}
		if err := filepath.WalkDir(dir, fn); err != nil {
			yield(nil, err)
		}
	}
}

// mediaDir returns the path to the media directory.
func (c *Collection) mediaDir() string {
	return mediaDir(c.dir)
}

// mediaPath returns the path to a media file.
func (c *Collection) mediaPath(name string) string {
	return filepath.Join(c.mediaDir(), name)
}

// Media is an interface for a media file.
type Media interface {
	Name() string
	Open() (io.ReadCloser, error)
}

// media implements the Media interface.
type media struct {
	name string
	path string
}

func (m *media) Name() string {
	return m.name
}

func (m *media) Open() (io.ReadCloser, error) {
	return os.Open(m.path)
}
