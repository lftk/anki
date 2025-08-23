package anki

import (
	"iter"
	"strings"
	"time"
)

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

type CardType int

const (
	CardTypeNew     CardType = 0
	CardTypeLearn   CardType = 1
	CardTypeReview  CardType = 2
	CardTypeRelearn CardType = 3
)

type CardQueue int

const (
	CardQueueNew           CardQueue = 0
	CardQueueLearn         CardQueue = 1
	CardQueueReview        CardQueue = 2
	CardQueueDayLearn      CardQueue = 3
	CardQueuePreviewRepeat CardQueue = 4
	CardQueueSuspended     CardQueue = -1
	CardQueueSchedBuried   CardQueue = -2
	CardQueueUserBuried    CardQueue = -3
)

func (c *Collection) GetCard(id int64) (*Card, error) {
	const query = `SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data FROM cards WHERE id = ?`

	return sqlGet(c.db, scanCard, query, id)
}

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
		card.Modified.UnixMilli(),
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

type ListCardsOptions struct {
	NoteID *int64
	DeckID *int64
}

func (c *Collection) ListCards(opts *ListCardsOptions) iter.Seq2[*Card, error] {
	var args []any
	var conds []string

	if opts != nil {
		if opts.NoteID != nil {
			conds = append(conds, "nid")
			args = append(args, *opts.NoteID)
		}

		if opts.DeckID != nil {
			conds = append(conds, "did")
			args = append(args, *opts.DeckID)
		}
	}

	query := getCardQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " = ? AND ") + " = ?"
	}

	return sqlSelectSeq(c.db, scanCard, query, args...)
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
