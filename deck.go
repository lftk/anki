package anki

import (
	"database/sql"
	"errors"
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

// NormalDeckKind creates a normal deck kind with the given configuration ID.
func NormalDeckKind(configID int64) *pb.DeckKind {
	return &pb.DeckKind{
		Kind: &pb.DeckKind_Normal{
			Normal: &pb.DeckNormal{
				ConfigId: configID,
			},
		},
	}
}

// DefaultDeckCommon returns the default deck common settings.
func DefaultDeckCommon() *pb.DeckCommon {
	return &pb.DeckCommon{
		StudyCollapsed:   true,
		BrowserCollapsed: true,
	}
}

// DeckName is the name of a deck.
type DeckName string

// JoinDeckName joins deck name components into a single DeckName.
// In Anki, deck names are hierarchical, separated by "::".
// Internally, they are stored with the U+001F INFORMATION SEPARATOR ONE character.
func JoinDeckName(components ...string) DeckName {
	return DeckName(strings.Join(components, deckNameSeparator))
}

// Parent returns the parent deck's name.
// If the deck is a top-level deck, it returns an empty string.
func (dn DeckName) Parent() DeckName {
	i := strings.LastIndex(string(dn), deckNameSeparator)
	if i != -1 {
		return dn[:i]
	}
	return ""
}

// Components returns the individual components of the deck name.
func (dn DeckName) Components() []string {
	return strings.Split(string(dn), deckNameSeparator)
}

// HumanString returns the deck name in a human-readable format,
// with "::" as the separator.
func (dn DeckName) HumanString() string {
	return strings.ReplaceAll(string(dn), deckNameSeparator, "::")
}

const deckNameSeparator = "\x1f"

// AddDeck adds a new deck to the collection.
// If the parent decks do not exist, they will be created automatically.
func (c *Collection) AddDeck(deck *Deck) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		var query = getDeckQuery + " WHERE name = ?"

		// Ensure all parent decks exist.
		for name := deck.Name.Parent(); name != ""; name = name.Parent() {
			_, err := sqlGet(tx, scanDeck, query, name)
			if err == nil {
				// Parent deck already exists.
				continue
			}

			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			// Create the parent deck if it doesn't exist.
			parent := &Deck{
				ID:       0, // Let the database assign an ID.
				Name:     name,
				Modified: time.Now(),
				USN:      deck.USN,
				Common:   deck.Common,
				Kind:     deck.Kind,
			}
			if err := addDeck(tx, parent); err != nil {
				return err
			}
		}

		return addDeck(tx, deck)
	})
}

// addDeck is a helper function to add a deck to the database.
func addDeck(e sqlExecer, deck *Deck) error {
	id := deck.ID
	if id == 0 {
		id = time.Now().UnixMilli()
	}

	if deck.Common == nil {
		deck.Common = DefaultDeckCommon()
	}
	common, err := proto.Marshal(deck.Common)
	if err != nil {
		return err
	}

	if deck.Kind == nil {
		deck.Kind = NormalDeckKind(1) // Use default deck config ID.
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
	id, err = sqlInsert(e, addDeckQuery, args...)
	if err == nil {
		deck.ID = id
	}
	return err
}

// GetDeck gets a deck by its ID.
func (c *Collection) GetDeck(id int64) (*Deck, error) {
	return getDeck(c.db, id)
}

// getDeck gets a deck by its ID.
func getDeck(q sqlQueryer, id int64) (*Deck, error) {
	return sqlGet(q, scanDeck, getDeckQuery+" WHERE id = ?", id)
}

// ListDecksOptions specifies options for listing decks.
type ListDecksOptions struct {
	ParentName *DeckName
}

// ListDecks lists all decks.
func (c *Collection) ListDecks(opts *ListDecksOptions) iter.Seq2[*Deck, error] {
	var args []any
	var conds []string

	if opts != nil {
		if opts.ParentName != nil && *opts.ParentName != "" {
			conds = append(conds, "name LIKE ? AND name NOT LIKE ?")
			pattern := string(*opts.ParentName) + deckNameSeparator + "%"
			args = append(args, pattern, pattern+deckNameSeparator+"%")
		}
	}

	query := getDeckQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	return sqlSelectSeq(c.db, scanDeck, query, args...)
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

// addDefaultDeck adds the default deck to the database.
func addDefaultDeck(e sqlExecer) error {
	return addDeck(e, &Deck{
		ID:       1,
		Name:     "Default",
		Modified: timeZero(),
		USN:      0,
		Common:   DefaultDeckCommon(),
		Kind:     NormalDeckKind(1), // Use default deck config ID.
	})
}
