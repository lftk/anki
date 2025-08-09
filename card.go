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
	row := c.db.QueryRow(`SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data FROM cards WHERE id = ?`, id)
	card := &Card{}
	var modMilli int64
	err := row.Scan(&card.ID, &card.NoteID, &card.DeckID, &card.Ordinal, &modMilli, &card.USN, &card.Type, &card.Queue, &card.Due, &card.Interval, &card.Factor, &card.Repetitions, &card.Lapses, &card.Left, &card.OriginalDue, &card.OriginalDeckID, &card.Flags, &card.Data)
	if err != nil {
		return nil, err
	}
	card.Modified = time.UnixMilli(modMilli)
	return card, nil
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
	_, err := c.db.Exec("DELETE FROM cards WHERE id = ?", id)
	return err
}

func (c *Collection) ListCards() iter.Seq2[*Card, error] {
	return func(yield func(*Card, error) bool) {
		rows, err := c.db.Query(`SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data FROM cards`)
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			card := &Card{}
			var modMilli int64
			if err := rows.Scan(&card.ID, &card.NoteID, &card.DeckID, &card.Ordinal, &modMilli, &card.USN, &card.Type, &card.Queue, &card.Due, &card.Interval, &card.Factor, &card.Repetitions, &card.Lapses, &card.Left, &card.OriginalDue, &card.OriginalDeckID, &card.Flags, &card.Data); err != nil {
				yield(nil, err)
				return
			}
			card.Modified = time.UnixMilli(modMilli)
			if !yield(card, nil) {
				return
			}
		}
	}
}
