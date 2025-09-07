package anki

import (
	"archive/zip"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lftk/anki/pb"
	"google.golang.org/protobuf/proto"
)

// Pack packs a collection into a zip file.
func Pack(w *zip.Writer, dir string) error {
	meta := &pb.PackageMetadata{
		Version: pb.PackageMetadata_VERSION_LATEST,
	}
	if err := writeMetadata(w, meta); err != nil {
		return err
	}
	if err := writeDatabase(w, databasePath(dir)); err != nil {
		return err
	}
	return writeMediaEntries(w, mediaDir(dir))
}

// Unpack unpacks a collection from a zip file.
func Unpack(r *zip.Reader, dir string) error {
	meta, err := detectMetadata(r)
	if err != nil {
		return err
	}
	if err = restoreDatabase(r, meta, databasePath(dir)); err != nil {
		return err
	}
	return restoreMediaEntries(r, meta, mediaDir(dir))
}

// isLegacyVersion checks if the package metadata is for a legacy version.
func isLegacyVersion(meta *pb.PackageMetadata) bool {
	return meta.Version == pb.PackageMetadata_VERSION_LEGACY_1 ||
		meta.Version == pb.PackageMetadata_VERSION_LEGACY_2
}

// zstdCompressed checks if the package is zstd compressed.
func zstdCompressed(meta *pb.PackageMetadata) bool {
	return !isLegacyVersion(meta)
}

// databaseName returns the database name based on the package metadata.
func databaseName(meta *pb.PackageMetadata) string {
	switch meta.Version {
	case pb.PackageMetadata_VERSION_LEGACY_1:
		return "collection.anki2"
	case pb.PackageMetadata_VERSION_LEGACY_2:
		return "collection.anki21"
	default: // PackageMetadata_VERSION_LATEST
		return "collection.anki21b"
	}
}

// databasePath returns the path to the database file.
func databasePath(dir string) string {
	return filepath.Join(dir, "collection.db")
}

// mediaDir returns the path to the media directory.
func mediaDir(dir string) string {
	return filepath.Join(dir, "media")
}

// restoreFile restores a file from a zip archive.
func restoreFile(r *zip.Reader, meta *pb.PackageMetadata, name string, path string) error {
	src, err := zipOpen(r, name, zstdCompressed(meta))
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close() //nolint:errcheck

	_, err = io.Copy(dst, src)
	return err
}

// restoreDatabase restores the database from a zip archive.
func restoreDatabase(r *zip.Reader, meta *pb.PackageMetadata, path string) error {
	return restoreFile(r, meta, databaseName(meta), path)
}

// writeDatabase writes the database to a zip archive.
func writeDatabase(w *zip.Writer, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck

	dst, err := zipCreate(w, "collection.anki21b", true)
	if err != nil {
		return err
	}
	defer dst.Close() //nolint:errcheck

	_, err = io.Copy(dst, src)
	return err
}

// restoreMediaEntries restores media entries from a zip archive.
func restoreMediaEntries(r *zip.Reader, meta *pb.PackageMetadata, dir string) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	media, err := readMediaEntries(r)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return err
	}
	for i, entry := range media.Entries {
		err = restoreFile(r, meta, fmt.Sprint(i), filepath.Join(dir, entry.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

// readMediaEntries reads media entries from a zip archive.
func readMediaEntries(r *zip.Reader) (*pb.MediaEntries, error) {
	b, err := zipReadAll(r, "media", true)
	if err != nil {
		return nil, err
	}
	var media pb.MediaEntries
	if err := proto.Unmarshal(b, &media); err != nil {
		return nil, err
	}
	return &media, nil
}

// writeMediaEntries writes media entries to a zip archive.
func writeMediaEntries(w *zip.Writer, dir string) error {
	var media pb.MediaEntries
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		sha1, err := writeMediaEntry(w, path, fmt.Sprint(len(media.Entries)))
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}

		media.Entries = append(media.Entries,
			&pb.MediaEntries_MediaEntry{
				Name: rel,
				Size: uint32(fi.Size()),
				Sha1: sha1,
			},
		)
		return nil
	}
	if err := filepath.WalkDir(dir, fn); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	b, err := proto.Marshal(&media)
	if err != nil {
		return err
	}
	return zipWrite(w, "media", true, b)
}

// writeMediaEntry writes a single media entry to a zip archive.
func writeMediaEntry(w *zip.Writer, path, name string) ([]byte, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close() //nolint:errcheck

	dst, err := zipCreate(w, name, true)
	if err != nil {
		return nil, err
	}
	defer dst.Close() //nolint:errcheck

	h := sha1.New()
	_, err = io.Copy(io.MultiWriter(h, dst), src)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// detectMetadata detects the package metadata from a zip archive.
func detectMetadata(r *zip.Reader) (*pb.PackageMetadata, error) {
	meta, err := readMetadata(r)
	if err == nil {
		return meta, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	ver := pb.PackageMetadata_VERSION_LEGACY_1
	if _, ok := zipLookup(r, "collection.anki21"); ok {
		ver = pb.PackageMetadata_VERSION_LEGACY_2
	}
	return &pb.PackageMetadata{Version: ver}, nil
}

// readMetadata reads the package metadata from a zip archive.
func readMetadata(r *zip.Reader) (*pb.PackageMetadata, error) {
	b, err := zipReadAll(r, "meta", false)
	if err != nil {
		return nil, err
	}
	var meta pb.PackageMetadata
	if err := proto.Unmarshal(b, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// writeMetadata writes the package metadata to a zip archive.
func writeMetadata(w *zip.Writer, meta *pb.PackageMetadata) error {
	b, err := proto.Marshal(meta)
	if err != nil {
		return err
	}
	return zipWrite(w, "meta", false, b)
}

// backup creates a backup of a collection.
func backup(src, dst string) error {
	if err := os.Mkdir(dst, 0755); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
	}

	if err := copyFile(databasePath(src), databasePath(dst)); err != nil {
		return err
	}

	return copyDir(mediaDir(src), mediaDir(dst))
}

// copyDir copies a directory.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)

		if d.IsDir() {
			err = os.MkdirAll(dstPath, 0755)
			if errors.Is(err, os.ErrExist) {
				err = nil
			}
			return err
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a file.
func copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close() //nolint:errcheck

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close() //nolint:errcheck

	_, err = io.Copy(w, r)
	return err
}
