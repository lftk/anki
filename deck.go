package anki

import (
	"iter"
	"strings"
	"time"

	"github.com/lftk/anki/pb"
	"google.golang.org/protobuf/proto"
)

// Deck represents a deck in Anki.
type Deck struct {
	ID       int64
	Name     DeckName
	Modified time.Time
	USN      int64
	Common   *pb.DeckCommon
	Kind     *pb.DeckKind
}

// DeckName is the name of a deck.
type DeckName string

// Parent returns the parent deck's name.
func (dn DeckName) Parent() DeckName {
	i := strings.LastIndexByte(string(dn), '\x1f')
	if i != -1 {
		return dn[:i]
	}
	return ""
}

// HumanString returns the deck name in a human-readable format.
func (dn DeckName) HumanString() string {
	return strings.ReplaceAll(string(dn), "\x1f", "::")
}

// AddDeck adds a new deck to the collection.
func (c *Collection) AddDeck(deck *Deck) error {
	id := deck.ID
	if id == 0 {
		id = time.Now().UnixMilli()
	}

	common, err := proto.Marshal(deck.Common)
	if err != nil {
		return err
	}

	kind, err := proto.Marshal(deck.Kind)
	if err != nil {
		return err
	}

	args := []any{
		id,
		deck.Name,
		timeUnix(deck.Modified),
		deck.USN,
		common,
		kind,
	}
	id, err = sqlInsert(c.db, addDeckQuery, args...)
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
	var common []byte
	var kind []byte
	if err := row.Scan(&deck.ID, &deck.Name, &mod, &deck.USN, &common, &kind); err != nil {
		return nil, err
	}

	deck.Common = new(pb.DeckCommon)
	if err := proto.Unmarshal(common, deck.Common); err != nil {
		return nil, err
	}

	deck.Kind = new(pb.DeckKind)
	if err := proto.Unmarshal(kind, deck.Kind); err != nil {
		return nil, err
	}

	deck.Modified = time.Unix(mod, 0)
	return &deck, nil
}
