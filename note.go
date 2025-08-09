package anki

import (
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"fmt"
	"iter"
	"regexp"
	"strings"
	"time"
)

const fieldSeparator = "\x1f"

var (
	htmlTagRegex   = regexp.MustCompile(`<[^>]*>`)
	mediaFileRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']?([^"'>]+)["']?[^>]*>|\[sound:([^\]]+)\]|\[anki:sound:([^\]]+)\]`)
)

type Note struct {
	ID         int64
	GUID       string
	NotetypeID int64
	Modified   time.Time
	USN        int64
	Tags       string
	Fields     []string
	Checksum   int64
	Flags      int64
	Data       string
}

func (c *Collection) GetNote(id int64) (*Note, error) {
	const query = `SELECT id, guid, mid, mod, usn, tags, flds, csum, flags, data FROM notes WHERE id = ?`

	return sqlQuery(c.db, scanNote, query, id)
}

func (c *Collection) AddNote(note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.addNote(note, notetype)
}

func (c *Collection) UpdateNote(note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.updateNote(note, notetype)
}

func (c *Collection) DeleteNote(id int64) error {
	_, err := c.db.Exec("DELETE FROM notes WHERE id = ?", id)
	return err
}

func (c *Collection) ListNotes() iter.Seq2[*Note, error] {
	const query = `SELECT id, guid, mid, mod, usn, tags, flds, csum, flags, data FROM notes`

	return sqlSelectSeq(c.db, scanNote, query)
}

func (c *Collection) addNote(note *Note, notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		if note.GUID == "" {
			var err error
			note.GUID, err = generateGUID()
			if err != nil {
				return fmt.Errorf("failed to generate guid: %w", err)
			}
		}
		note.Modified = time.Now()
		note.USN = -1

		sortField, keyField, err := c.prepareNoteFields(note, notetype)
		if err != nil {
			return err
		}
		note.Checksum = calculateChecksum(keyField)

		flds := strings.Join(note.Fields, fieldSeparator)

		res, err := tx.Exec(`INSERT INTO notes (guid, mid, mod, usn, tags, flds, sfld, csum, flags, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			note.GUID, note.NotetypeID, note.Modified.UnixMilli(), note.USN, note.Tags, flds, sortField, note.Checksum, note.Flags, note.Data)
		if err != nil {
			return err
		}

		note.ID, err = res.LastInsertId()
		return err
	})
}

func (c *Collection) updateNote(note *Note, notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		now := time.Now()
		note.Modified = now
		note.USN = -1

		sortField, keyField, err := c.prepareNoteFields(note, notetype)
		if err != nil {
			return err
		}
		note.Checksum = calculateChecksum(keyField)

		flds := strings.Join(note.Fields, fieldSeparator)

		_, err = tx.Exec(`UPDATE notes SET guid = ?, mid = ?, mod = ?, usn = ?, tags = ?, flds = ?, sfld = ?, csum = ?, flags = ?, data = ? WHERE id = ?`,
			note.GUID, note.NotetypeID, note.Modified.UnixMilli(), note.USN, note.Tags, flds, sortField, note.Checksum, note.Flags, note.Data, note.ID)
		return err
	})
}

func (c *Collection) prepareNoteFields(note *Note, notetype *Notetype) (sortField string, keyField string, err error) {
	if len(note.Fields) == 0 {
		err = fmt.Errorf("cannot process note with no fields")
		return
	}

	keyField = stripHTML(note.Fields[0])

	sortIndex := notetype.Config.GetSortFieldIdx()
	if sortIndex == 0 {
		sortField = keyField
	} else {
		if int(sortIndex) >= len(note.Fields) {
			err = fmt.Errorf("sort_field_idx %d is out of bounds for note with %d fields", sortIndex, len(note.Fields))
			return
		}
		sortField = stripHTML(note.Fields[sortIndex])
	}
	return
}

func calculateChecksum(keyField string) int64 {
	hasher := sha1.New()
	hasher.Write([]byte(keyField))
	hashBytes := hasher.Sum(nil)
	return int64(int32(binary.BigEndian.Uint32(hashBytes[:4])))
}

func stripHTML(s string) string {
	result := mediaFileRegex.ReplaceAllStringFunc(s, func(match string) string {
		submatches := mediaFileRegex.FindStringSubmatch(match)
		for i := 1; i < len(submatches); i++ {
			if submatches[i] != "" {
				return submatches[i]
			}
		}
		return ""
	})

	result = htmlTagRegex.ReplaceAllString(result, "")

	return result
}

func scanNote(_ sqlQueryer, row sqlRow) (*Note, error) {
	var note Note
	var mod int64
	var flds string

	dest := []any{
		&note.ID,
		&note.GUID,
		&note.NotetypeID,
		&mod,
		&note.USN,
		&note.Tags,
		&flds,
		&note.Checksum,
		&note.Flags,
		&note.Data,
	}
	err := row.Scan(dest...)
	if err != nil {
		return nil, err
	}

	note.Modified = time.UnixMilli(mod)
	note.Fields = strings.Split(flds, fieldSeparator)

	return &note, nil
}
