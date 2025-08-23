package anki

import (
	"fmt"
	"iter"
	"regexp"
	"slices"

	"github.com/lftk/anki/pb"
)

func generateCards(deckID int64, note *Note, notetype *Notetype) iter.Seq2[*Card, error] {
	return func(yield func(*Card, error) bool) {
		cards, err := newCardsRequired(deckID, note, notetype)
		if err != nil {
			yield(nil, err)
			return
		}
		for _, card := range cards {
			c := &Card{
				NoteID:   note.ID,
				DeckID:   card.DeckID,
				Ordinal:  card.Ordinal,
				Modified: timeZero(),
				USN:      -1,
				Type:     CardTypeNew,
				Queue:    CardQueueNew,
			}
			if !yield(c, nil) {
				return
			}
		}
	}
}

func newCardsRequired(deckID int64, note *Note, notetype *Notetype) ([]*cardToGenerate, error) {
	switch notetype.Config.Kind {
	case pb.NotetypeConfig_KIND_NORMAL:
		return newCardsRequiredNormal(deckID, note, notetype)
	case pb.NotetypeConfig_KIND_CLOZE:
		return newCardsRequiredCloze(deckID, note)
	default:
		return nil, fmt.Errorf("invalid or unsupported notetype kind: %s", notetype.Config.Kind)
	}
}

func newCardsRequiredNormal(deckID int64, note *Note, notetype *Notetype) ([]*cardToGenerate, error) {
	fields := nonemptyFields(note, notetype)
	cards := make([]*cardToGenerate, 0, len(notetype.Templates))
	for ord, template := range notetype.Templates {
		ok, err := rendersTemplate(template, fields)
		if err != nil {
			return nil, err
		}
		if ok {
			targetDeckID := template.Config.TargetDeckId
			if targetDeckID == 0 {
				targetDeckID = deckID
			}
			card := &cardToGenerate{
				Ordinal: int64(ord),
				DeckID:  targetDeckID,
				Due:     0,
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

func newCardsRequiredCloze(deckID int64, note *Note) ([]*cardToGenerate, error) {
	ords, err := clozeNumberInFields(note.Fields)
	if err != nil {
		return nil, err
	}
	cards := make([]*cardToGenerate, 0, len(ords))
	for _, ord := range ords {
		card := &cardToGenerate{
			Ordinal: int64(ord - 1),
			DeckID:  deckID,
			Due:     0,
		}
		cards = append(cards, card)
	}
	return cards, nil
}

type cardToGenerate struct {
	Ordinal int64
	DeckID  int64
	Due     int64
}
