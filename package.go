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

	"google.golang.org/protobuf/proto"

	"github.com/lftk/anki/internal/pb"
)

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

func isLegacyVersion(meta *pb.PackageMetadata) bool {
	return meta.Version == pb.PackageMetadata_VERSION_LEGACY_1 ||
		meta.Version == pb.PackageMetadata_VERSION_LEGACY_2
}

func zstdCompressed(meta *pb.PackageMetadata) bool {
	return !isLegacyVersion(meta)
}

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

func databasePath(dir string) string {
	return filepath.Join(dir, "collection.db")
}

func mediaDir(dir string) string {
	return filepath.Join(dir, "media")
}

func restoreFile(r *zip.Reader, meta *pb.PackageMetadata, name string, path string) error {
	src, err := zipOpen(r, name, zstdCompressed(meta))
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func restoreDatabase(r *zip.Reader, meta *pb.PackageMetadata, path string) error {
	return restoreFile(r, meta, databaseName(meta), path)
}

func writeDatabase(w *zip.Writer, path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := zipCreate(w, "collection.anki21b", true)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func restoreMediaEntries(r *zip.Reader, meta *pb.PackageMetadata, dir string) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	media, err := readMediaEntries(r)
	if err != nil {
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
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
		return err
	}

	b, err := proto.Marshal(&media)
	if err != nil {
		return err
	}
	return zipWrite(w, "media", true, b)
}

func writeMediaEntry(w *zip.Writer, path, name string) ([]byte, error) {
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dst, err := zipCreate(w, name, true)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	h := sha1.New()
	_, err = io.Copy(io.MultiWriter(h, dst), src)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

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

func writeMetadata(w *zip.Writer, meta *pb.PackageMetadata) error {
	b, err := proto.Marshal(meta)
	if err != nil {
		return err
	}
	return zipWrite(w, "meta", false, b)
}

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

func copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}
