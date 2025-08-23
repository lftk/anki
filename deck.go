package anki

import (
	"iter"
	"time"
)

// Deck represents a deck in Anki.
type Deck struct {
	ID       int64
	Name     string
	Modified time.Time
	USN      int64
	Common   []byte
	Kind     []byte
}

// AddDeck adds a new deck to the collection.
func (c *Collection) AddDeck(deck *Deck) error {
	id := deck.ID
	if id == 0 {
		id = time.Now().UnixMilli()
	}
	args := []any{
		id,
		deck.Name,
		timeUnix(deck.Modified),
		deck.USN,
		deck.Common,
		deck.Kind,
	}
	id, err := sqlInsert(c.db, addDeckQuery, args...)
	if err == nil {
		deck.ID = id
	}
	return err
}

// GetDeck gets a deck by its ID.
func (c *Collection) GetDeck(id int64) (*Deck, error) {
	return sqlGet(c.db, scanDeck, getDeckQuery+" WHERE id = ?", id)
}

// ListDecksOptions specifies options for listing decks.
type ListDecksOptions struct{}

// ListDecks lists all decks.
func (c *Collection) ListDecks(*ListDecksOptions) iter.Seq2[*Deck, error] {
	return sqlSelectSeq(c.db, scanDeck, getDeckQuery)
}

// scanDeck scans a deck from a database row.
func scanDeck(_ sqlQueryer, row sqlRow) (*Deck, error) {
	var deck Deck
	var mod int64
	if err := row.Scan(&deck.ID, &deck.Name, &mod, &deck.USN, &deck.Common, &deck.Kind); err != nil {
		return nil, err
	}
	deck.Modified = time.Unix(mod, 0)
	return &deck, nil
}
