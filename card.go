package anki

import (
	"database/sql"
	"iter"
	"time"
)

type Card struct {
	ID             int64
	NoteID         int64
	DeckID         int64
	Ordinal        int64
	Modified       time.Time
	USN            int64
	Type           int64
	Queue          int64
	Due            int64
	Interval       int64
	Factor         int64
	Repetitions    int64
	Lapses         int64
	Left           int64
	OriginalDue    int64
	OriginalDeckID int64
	Flags          int64
	Data           string
}

func (c *Collection) GetCard(id int64) (*Card, error) {
	const query = `SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data FROM cards WHERE id = ?`

	return sqlGet(c.db, scanCard, query, id)
}

func (c *Collection) AddCard(card *Card) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		card.Modified = time.Now()
		card.USN = -1

		res, err := tx.Exec(`INSERT INTO cards (nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			card.NoteID, card.DeckID, card.Ordinal, card.Modified.UnixMilli(), card.USN, card.Type, card.Queue, card.Due, card.Interval, card.Factor, card.Repetitions, card.Lapses, card.Left, card.OriginalDue, card.OriginalDeckID, card.Flags, card.Data)
		if err != nil {
			return err
		}

		card.ID, err = res.LastInsertId()
		return err
	})
}

func (c *Collection) UpdateCard(card *Card) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		card.Modified = time.Now()
		card.USN = -1

		_, err := tx.Exec(`UPDATE cards SET nid = ?, did = ?, ord = ?, mod = ?, usn = ?, type = ?, queue = ?, due = ?, ivl = ?, factor = ?, reps = ?, lapses = ?, left = ?, odue = ?, odid = ?, flags = ?, data = ? WHERE id = ?`,
			card.NoteID, card.DeckID, card.Ordinal, card.Modified.UnixMilli(), card.USN, card.Type, card.Queue, card.Due, card.Interval, card.Factor, card.Repetitions, card.Lapses, card.Left, card.OriginalDue, card.OriginalDeckID, card.Flags, card.Data, card.ID)
		return err
	})
}

func (c *Collection) DeleteCard(id int64) error {
	return sqlExecute(c.db, "DELETE FROM cards WHERE id = ?", id)
}

func (c *Collection) ListCards() iter.Seq2[*Card, error] {
	const query = `SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data FROM cards`

	return sqlSelectSeq(c.db, scanCard, query)
}

func scanCard(_ sqlQueryer, row sqlRow) (*Card, error) {
	var card Card
	var mod int64

	dest := []any{
		&card.ID,
		&card.NoteID,
		&card.DeckID,
		&card.Ordinal,
		&mod,
		&card.USN,
		&card.Type,
		&card.Queue,
		&card.Due,
		&card.Interval,
		&card.Factor,
		&card.Repetitions,
		&card.Lapses,
		&card.Left,
		&card.OriginalDue,
		&card.OriginalDeckID,
		&card.Flags,
		&card.Data,
	}
	err := row.Scan(dest...)
	if err != nil {
		return nil, err
	}

	card.Modified = time.UnixMilli(mod)

	return &card, nil
}

func deleteCards(e sqlExecer, noteID int64) error {
	const query = `DELETE FROM cards WHERE nid = ?`

	return sqlExecute(e, query, noteID)
}
