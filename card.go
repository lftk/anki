package anki

import (
	"iter"
	"strings"
	"time"
)

// Card represents a card in Anki.
type Card struct {
	ID             int64
	NoteID         int64
	DeckID         int64
	Ordinal        int64
	Modified       time.Time
	USN            int64
	Type           CardType
	Queue          CardQueue
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

// CardType represents the type of a card.
type CardType int

const (
	// CardTypeNew is a new card.
	CardTypeNew CardType = 0
	// CardTypeLearn is a learning card.
	CardTypeLearn CardType = 1
	// CardTypeReview is a review card.
	CardTypeReview CardType = 2
	// CardTypeRelearn is a relearning card.
	CardTypeRelearn CardType = 3
)

// CardQueue represents the queue of a card.
type CardQueue int

const (
	// CardQueueNew is the new card queue.
	CardQueueNew CardQueue = 0
	// CardQueueLearn is the learning queue.
	CardQueueLearn CardQueue = 1
	// CardQueueReview is the review queue.
	CardQueueReview CardQueue = 2
	// CardQueueDayLearn is the day learning queue.
	CardQueueDayLearn CardQueue = 3
	// CardQueuePreviewRepeat is the preview repeat queue.
	CardQueuePreviewRepeat CardQueue = 4
	// CardQueueSuspended is the suspended queue.
	CardQueueSuspended CardQueue = -1
	// CardQueueSchedBuried is the scheduler buried queue.
	CardQueueSchedBuried CardQueue = -2
	// CardQueueUserBuried is the user buried queue.
	CardQueueUserBuried CardQueue = -3
)

// GetCard gets a card by its ID.
func (c *Collection) GetCard(id int64) (*Card, error) {
	return sqlGet(c.db, scanCard, getCardQuery+" WHERE id = ?", id)
}

// addCard adds a new card to the collection.
func addCard(e sqlExecer, card *Card) error {
	id := card.ID
	if id == 0 {
		id = time.Now().UnixMilli()
	}
	args := []any{
		id,
		card.NoteID,
		card.DeckID,
		card.Ordinal,
		timeUnix(card.Modified),
		card.USN,
		card.Type,
		card.Queue,
		card.Due,
		card.Interval,
		card.Factor,
		card.Repetitions,
		card.Lapses,
		card.Left,
		card.OriginalDue,
		card.OriginalDeckID,
		card.Flags,
		card.Data,
	}
	id, err := sqlInsert(e, addCardQuery, args...)
	if err == nil {
		card.ID = id
	}
	return err
}

// ListCardsOptions specifies options for listing cards.
type ListCardsOptions struct {
	NoteID *int64
	DeckID *int64
}

// ListCards lists cards with optional filtering.
func (c *Collection) ListCards(opts *ListCardsOptions) iter.Seq2[*Card, error] {
	var args []any
	var conds []string

	if opts != nil {
		if opts.NoteID != nil {
			conds = append(conds, "nid = ?")
			args = append(args, *opts.NoteID)
		}

		if opts.DeckID != nil {
			conds = append(conds, "did = ?")
			args = append(args, *opts.DeckID)
		}
	}

	query := getCardQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	return sqlSelectSeq(c.db, scanCard, query, args...)
}

// scanCard scans a card from a database row.
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

	card.Modified = time.Unix(mod, 0)

	return &card, nil
}

// deleteCards deletes all cards for a given note.
func deleteCards(e sqlExecer, noteID int64) error {
	return sqlExecute(e, deleteCardsQuery, noteID)
}
