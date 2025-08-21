package anki

import (
	"crypto/sha1"
	"database/sql"
	"encoding/binary"
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lftk/anki/internal/pb"
)

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

func (c *Collection) GetNote(id int64) (*Note, error) {
	const query = `SELECT id, guid, mid, mod, usn, tags, flds, csum, flags, data FROM notes WHERE id = ?`

	return sqlGet(c.db, scanNote, query, id)
}

func (c *Collection) AddNote(deckID int64, note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.addNote(deckID, note, notetype)
}

func (c *Collection) UpdateNote(note *Note) error {
	notetype, err := c.GetNotetype(note.NotetypeID)
	if err != nil {
		return err
	}
	return c.updateNote(note, notetype)
}

func (c *Collection) DeleteNote(id int64) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		return deleteNote(tx, id)
	})
}

func deleteNotes(e sqlExt, notetypeID int64) error {
	const query = `SELECT id FROM notes WHERE mid = ?`

	for id, err := range sqlSelectSeq(e, scanValue[int64], query, notetypeID) {
		if err != nil {
			return err
		}
		if err := deleteNote(e, id); err != nil {
			return err
		}
	}
	return nil
}

func deleteNote(e sqlExecer, noteID int64) error {
	const query = `DELETE FROM notes WHERE id = ?`

	if err := sqlExecute(e, query, noteID); err != nil {
		return err
	}
	return deleteCards(e, noteID)
}

func (c *Collection) ListNotes() iter.Seq2[*Note, error] {
	const query = `SELECT id, guid, mid, mod, usn, tags, flds, csum, flags, data FROM notes`

	return sqlSelectSeq(c.db, scanNote, query)
}

func (c *Collection) addNote(deckID int64, note *Note, notetype *Notetype) error {
	const query = `
INSERT INTO
  notes (
    id,
    guid,
    mid,
    mod,
    usn,
    tags,
    flds,
    sfld,
    csum,
    flags,
    data
  )
VALUES
  (
    (
      CASE
        WHEN ?1 IN (
          SELECT
            id
          FROM
            notes
        ) THEN (
          SELECT
            max(id) + 1
          FROM
            notes
        )
        ELSE ?1
      END
    ),
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?, -- 0
    ?, -- ""
  )
`

	return sqlTransact(c.db, func(tx *sql.Tx) error {
		if note.GUID == "" {
			var err error
			note.GUID, err = randomGUID()
			if err != nil {
				return err
			}
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
			note.Modified.UnixMilli(),
			note.USN,
			joinTags(note.Tags),
			joinFields(note.Fields),
			sfld,
			note.Checksum,
			note.Flags,
			note.Data,
		}
		note.ID, err = sqlInsert(tx, query, args...)
		if err != nil {
			return err
		}

		cards, err := newCardsRequired(deckID, note, notetype)
		if err != nil {
			return err
		}

		for _, card := range cards {
			if err = addCard(tx, card); err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Collection) updateNote(note *Note, notetype *Notetype) error {
	const query = `UPDATE notes SET guid = ?, mid = ?, mod = ?, usn = ?, tags = ?, flds = ?, sfld = ?, csum = ?, flags = ?, data = ? WHERE id = ?`

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
			note.Modified.UnixMilli(),
			note.USN,
			joinTags(note.Tags),
			joinFields(note.Fields),
			sfld,
			note.Checksum,
			note.Flags,
			note.Data,
			note.ID,
		}
		return sqlExecute(tx, query, args...)
	})
}

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

	note.Modified = time.UnixMilli(mod)
	note.Tags = splitTags(tags)
	note.Fields = splitFields(fields)

	return &note, nil
}

func randomGUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return " " + strings.Join(tags, " ") + " "
}

func splitTags(tags string) []string {
	// TODO: prefix and suffix spaces
	return strings.FieldsFunc(tags, isTagSeparator)
}

func isTagSeparator(r rune) bool {
	return r == ' ' || r == '\u3000'
}

const fieldSeparator = "\x1f"

func joinFields(fields []string) string {
	return strings.Join(fields, fieldSeparator)
}

func splitFields(fields string) []string {
	return strings.Split(fields, fieldSeparator)
}

func newCardsRequired(deckID int64, note *Note, notetype *Notetype) ([]*Card, error) {
	switch notetype.Config.Kind {
	case pb.NotetypeConfig_KIND_NORMAL:
		return newCardsRequiredNormal(note, notetype)
	case pb.NotetypeConfig_KIND_CLOZE:
		// todo
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid or unsupported notetype kind: %s", notetype.Config.Kind)
	}
}

func newCardsRequiredNormal(note *Note, notetype *Notetype) ([]*Card, error) {
	// fields := nonemptyFields(note, notetype)
	cards := make([]*Card, 0, len(notetype.Templates))
	for ord, template := range notetype.Templates {
		ok, err := rendersTemplate(template, nil) // todo
		if err != nil {
			return nil, err
		}
		if ok {
			card := &Card{
				NoteID:   note.ID,
				DeckID:   template.Config.TargetDeckId,
				Ordinal:  int64(ord),
				Modified: timeZero(),
				USN:      -1,
				Type:     CardTypeNew,
				Queue:    CardQueueNew,
			}
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func nonemptyFields(note *Note, notetype *Notetype) []string {
	fields := make([]string, 0, len(note.Fields))
	for ord, field := range note.Fields {
		if !fieldIsEmpty(field) {
			fn := func(f *Field) bool {
				return f.Ordinal == ord
			}
			if i := slices.IndexFunc(notetype.Fields, fn); i != -1 {
				fields = append(fields, notetype.Fields[i].Name)
			}
		}
	}
	return fields
}

var fieldIsEmptyRe = regexp.MustCompile(`(?i)^(?:[\s]|</?(?:br|div)\s*/?>)*$`)

// fieldIsEmpty returns true if the provided text contains only whitespace and/or empty BR/DIV tags.
func fieldIsEmpty(text string) bool {
	return fieldIsEmptyRe.MatchString(text)
}
