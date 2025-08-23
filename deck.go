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
	return sqlGet(c.db, scanDeck, getDeckQuery+" WHERE id = ?", id)
}

type ListDecksOptions struct{}

func (c *Collection) ListDecks(opts *ListDecksOptions) iter.Seq2[*Deck, error] {
	return sqlSelectSeq(c.db, scanDeck, getDeckQuery)
}

func scanDeck(_ sqlQueryer, row sqlRow) (*Deck, error) {
	var deck Deck
	var mod int64
	if err := row.Scan(&deck.ID, &deck.Name, &mod, &deck.USN, &deck.Common, &deck.Kind); err != nil {
		return nil, err
	}
	deck.Modified = time.Unix(mod, 0)
	return &deck, nil
}
