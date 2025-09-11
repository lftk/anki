package anki

import (
	"fmt"
	"iter"
	"regexp"
	"slices"

	"github.com/lftk/anki/pb"
)

// generateCards generates cards for a given note.
// It takes a deckID, a note, and a notetype and returns a sequence of cards.
func generateCards(deckID int64, note *Note, notetype *Notetype, existingOrds []int) iter.Seq2[*Card, error] {
	return func(yield func(*Card, error) bool) {
		cards, err := newCardsRequired(deckID, note, notetype)
		if err != nil {
			yield(nil, err)
			return
		}
		for _, card := range cards {
			if slices.Contains(existingOrds, card.Ordinal) {
				continue
			}
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

// newCardsRequired determines which cards need to be generated for a note based on the notetype.
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

// newCardsRequiredNormal handles card generation for normal notetypes.
// It checks which templates render to a non-empty card and creates a card for each.
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
				Ordinal: ord,
				DeckID:  targetDeckID,
				Due:     0,
			}
			cards = append(cards, card)
		}
	}
	return cards, nil
}

// nonemptyFields returns a list of field names that are not empty for a given note.
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

// newCardsRequiredCloze handles card generation for cloze notetypes.
// It finds all the cloze deletions in the note's fields and creates a card for each.
func newCardsRequiredCloze(deckID int64, note *Note) ([]*cardToGenerate, error) {
	ords, err := clozeNumberInFields(note.Fields)
	if err != nil {
		return nil, err
	}
	cards := make([]*cardToGenerate, 0, len(ords))
	for _, ord := range ords {
		card := &cardToGenerate{
			Ordinal: ord - 1,
			DeckID:  deckID,
			Due:     0,
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// cardToGenerate is a struct that holds information about a card to be generated.
type cardToGenerate struct {
	Ordinal int
	DeckID  int64
	Due     int64
}
