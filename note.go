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

// Note represents a note in Anki.
type Note struct {
	ID         int64
	GUID       string
	NotetypeID int64
	Modified   time.Time
	USN        int64
	Tags       []string
	Fields     []string
	Checksum   int64
	Flags      int64
	Data       string
}

// GetNote gets a note by its ID.
func (c *Collection) GetNote(id int64) (*Note, error) {
	return sqlGet(c.db, scanNote, getNoteQuery+" WHERE id = ?", id)
}

// AddNote adds a new note to the collection.
func (c *Collection) AddNote(deckID int64, note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.addNote(deckID, note, notetype)
}

// UpdateNote updates an existing note in the collection.
func (c *Collection) UpdateNote(note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.updateNote(note, notetype)
}

// DeleteNote deletes a note from the collection by its ID.
func (c *Collection) DeleteNote(id int64) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		return deleteNote(tx, id)
	})
}

// deleteNotes deletes all notes of a given notetype.
func deleteNotes(e sqlExt, notetypeID int64) error {
	for id, err := range sqlSelectSeq(e, scanValue[int64], listNoteIDsQuery, notetypeID) {
		if err != nil {
			return err
		}
		if err := deleteNote(e, id); err != nil {
			return err
		}
	}
	return nil
}

// deleteNote deletes a note and its associated cards.
func deleteNote(e sqlExecer, noteID int64) error {
	if err := sqlExecute(e, deleteNoteQuery, noteID); err != nil {
		return err
	}
	return deleteCards(e, noteID)
}

// ListNotesOptions specifies options for listing notes.
type ListNotesOptions struct {
	NotetypeID *int64
}

// ListNotes lists notes with optional filtering.
func (c *Collection) ListNotes(opts *ListNotesOptions) iter.Seq2[*Note, error] {
	var args []any
	var conds []string

	if opts != nil {
		if opts.NotetypeID != nil {
			conds = append(conds, "mid = ?")
			args = append(args, *opts.NotetypeID)
		}
	}

	query := getNoteQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	return sqlSelectSeq(c.db, scanNote, query, args...)
}

// addNote is an internal helper to add a note.
func (c *Collection) addNote(deckID int64, note *Note, notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		if note.GUID == "" {
			guid, err := randomGUID()
			if err != nil {
				return err
			}
			note.GUID = guid
		}
		note.Modified = time.Now()
		note.USN = -1

		fld1, sfld, err := prepareNoteFields(note, notetype)
		if err != nil {
			return err
		}
		note.Checksum = fieldChecksum(fld1)

		id := note.ID
		if id == 0 {
			id = time.Now().UnixMilli()
		}

		args := []any{
			id,
			note.GUID,
			note.NotetypeID,
			timeUnix(note.Modified),
			note.USN,
			joinTags(note.Tags),
			joinFields(note.Fields),
			sfld,
			note.Checksum,
			note.Flags,
			note.Data,
		}
		note.ID, err = sqlInsert(tx, addNoteQuery, args...)
		if err != nil {
			return err
		}

		for card, err := range generateCards(deckID, note, notetype) {
			if err != nil {
				return err
			}
			if err = addCard(tx, card); err != nil {
				return err
			}
		}
		return nil
	})
}

// updateNote is an internal helper to update a note.
func (c *Collection) updateNote(note *Note, notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		note.Modified = time.Now()
		note.USN = -1

		fld1, sfld, err := prepareNoteFields(note, notetype)
		if err != nil {
			return err
		}
		note.Checksum = fieldChecksum(fld1)

		args := []any{
			note.GUID,
			note.NotetypeID,
			timeUnix(note.Modified),
			note.USN,
			joinTags(note.Tags),
			joinFields(note.Fields),
			sfld,
			note.Checksum,
			note.Flags,
			note.Data,
			note.ID,
		}
		return sqlExecute(tx, updateNoteQuery, args...)
	})
}

// prepareNoteFields prepares the first field and sort field for a note.
func prepareNoteFields(note *Note, notetype *Notetype) (fld1, sfld string, err error) {
	if len(note.Fields) == 0 {
		err = fmt.Errorf("cannot process note with no fields")
		return
	}
	fld1 = stripHTML(note.Fields[0])
	sortIndex := notetype.Config.GetSortFieldIdx()
	if sortIndex == 0 {
		sfld = fld1
	} else {
		if int(sortIndex) >= len(note.Fields) {
			err = fmt.Errorf("sort_field_idx %d is out of bounds for note with %d fields", sortIndex, len(note.Fields))
			return
		}
		sfld = stripHTML(note.Fields[sortIndex])
	}
	return
}

// fieldChecksum calculates the checksum of a field.
func fieldChecksum(field string) int64 {
	h := sha1.New()
	h.Write([]byte(field))
	sum := h.Sum(nil)
	return int64(int32(binary.BigEndian.Uint32(sum[:4])))
}

var (
	htmlTagRegex   = regexp.MustCompile(`<[^>]*>`)
	mediaFileRegex = regexp.MustCompile(`(?i)<img[^>]+src=["']?([^"'>]+)["']?[^>]*>|\[sound:([^\]]+)\]|\[anki:sound:([^\]]+)\]`)
)

// stripHTML strips HTML tags from a string, preserving media file references.
func stripHTML(s string) string {
	repl := func(match string) string {
		submatches := mediaFileRegex.FindStringSubmatch(match)
		for i := 1; i < len(submatches); i++ {
			if submatches[i] != "" {
				return submatches[i]
			}
		}
		return ""
	}
	return htmlTagRegex.ReplaceAllString(mediaFileRegex.ReplaceAllStringFunc(s, repl), "")
}

// scanNote scans a note from a database row.
func scanNote(_ sqlQueryer, row sqlRow) (*Note, error) {
	var note Note
	var mod int64
	var tags string
	var fields string

	dest := []any{
		&note.ID,
		&note.GUID,
		&note.NotetypeID,
		&mod,
		&note.USN,
		&tags,
		&fields,
		&note.Checksum,
		&note.Flags,
		&note.Data,
	}
	err := row.Scan(dest...)
	if err != nil {
		return nil, err
	}

	note.Modified = time.Unix(mod, 0)
	note.Tags = splitTags(tags)
	note.Fields = splitFields(fields)

	return &note, nil
}

// joinTags joins a slice of tags into a single string.
func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return " " + strings.Join(tags, " ") + " "
}

// splitTags splits a string of tags into a slice.
func splitTags(tags string) []string {
	return strings.FieldsFunc(tags, isTagSeparator)
}

// isTagSeparator checks if a rune is a tag separator.
func isTagSeparator(r rune) bool {
	return r == ' ' || r == '\u3000'
}

const fieldSeparator = "\x1f"

// joinFields joins a slice of fields into a single string.
func joinFields(fields []string) string {
	return strings.Join(fields, fieldSeparator)
}

// splitFields splits a string of fields into a slice.
func splitFields(fields string) []string {
	return strings.Split(fields, fieldSeparator)
}
