package anki

import (
	"iter"
	"time"
)

type Deck struct {
	ID       int64
	Name     string
	Modified time.Time
	USN      int64
	Common   []byte
	Kind     []byte
}

func (c *Collection) GetDeck(id int64) (*Deck, error) {
	deck := &Deck{}
	var modSecs int64
	err := c.db.QueryRow("SELECT id, name, mtime_secs, usn, common, kind FROM decks WHERE id = ?", id).Scan(
		&deck.ID, &deck.Name, &modSecs, &deck.USN, &deck.Common, &deck.Kind)
	if err != nil {
		return nil, err
	}
	deck.Modified = time.Unix(modSecs, 0)
	return deck, err
}

func (c *Collection) Decks() iter.Seq2[*Deck, error] {
	return func(yield func(*Deck, error) bool) {
		rows, err := c.db.Query("SELECT id, name, mtime_secs, usn, common, kind FROM decks")
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			deck := &Deck{}
			var modSecs int64
			if err := rows.Scan(&deck.ID, &deck.Name, &modSecs, &deck.USN, &deck.Common, &deck.Kind); err != nil {
				yield(nil, err)
				return
			}
			deck.Modified = time.Unix(modSecs, 0)
			if !yield(deck, nil) {
				return
			}
		}
	}
}
