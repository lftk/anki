package anki

import (
	"database/sql"
	"iter"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/lftk/anki/pb"
	"google.golang.org/protobuf/proto"
)

// Notetype represents a notetype in Anki.
type Notetype struct {
	ID        int64
	Name      string
	Modified  time.Time
	USN       int64
	Fields    []*Field
	Templates []*Template
	Config    *pb.NotetypeConfig
}

// NewNotetypeConfig creates a new notetype configuration.
// If cloze is true, the notetype is configured for cloze deletion.
func NewNotetypeConfig(css string, cloze bool) *pb.NotetypeConfig {
	kind := pb.NotetypeConfig_KIND_NORMAL
	if cloze {
		kind = pb.NotetypeConfig_KIND_CLOZE
	}
	return &pb.NotetypeConfig{
		Css:  css,
		Kind: kind,
	}
}

// Field represents a field in a notetype.
type Field struct {
	Ordinal int
	Name    string
	Config  *pb.FieldConfig
}

// NewField creates a new field with the given name and default configuration.
// The ordinal is initialized to -1 and will be set when added to a notetype.
func NewField(name string) *Field {
	id := time.Now().UnixMilli()
	return &Field{
		Ordinal: -1,
		Name:    name,
		Config: &pb.FieldConfig{
			Id:                &id,
			Sticky:            false,
			Rtl:               false,
			PlainText:         false,
			FontName:          "Arial",
			FontSize:          20,
			Description:       "",
			Collapsed:         false,
			ExcludeFromSearch: false,
			Tag:               nil,
			PreventDeletion:   false,
			Other:             nil,
		},
	}
}

// Template represents a template in a notetype.
type Template struct {
	Ordinal  int
	Name     string
	Modified time.Time
	USN      int64
	Config   *pb.TemplateConfig
}

// NewTemplate creates a new template with the given name, question format,
// and answer format.
// The ordinal is initialized to -1 and will be set when added to a notetype.
func NewTemplate(name, qfmt, afmt string) *Template {
	id := time.Now().UnixMilli()
	return &Template{
		Ordinal:  -1,
		Name:     name,
		Modified: timeZero(),
		USN:      0,
		Config: &pb.TemplateConfig{
			Id:      &id,
			QFormat: qfmt,
			AFormat: afmt,
		},
	}
}

// GetNotetype gets a notetype by its ID.
func (c *Collection) GetNotetype(id int64) (*Notetype, error) {
	return getNotetype(c.db, id)
}

// AddNotetype adds a new notetype to the collection.
func (c *Collection) AddNotetype(notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		id := notetype.ID
		if id == 0 {
			id = time.Now().UnixMilli()
		}

		notetype.Modified = time.Now()
		notetype.USN = -1

		if notetype.Config == nil {
			notetype.Config = &pb.NotetypeConfig{}
		}
		config, err := proto.Marshal(notetype.Config)
		if err != nil {
			return err
		}

		args := []any{
			id,
			notetype.Name,
			timeUnix(notetype.Modified),
			notetype.USN,
			config,
		}
		notetype.ID, err = sqlInsert(tx, addNotetypeQuery, args...)
		if err != nil {
			return err
		}

		return addFieldsAndTemplates(tx, notetype)
	})
}

// UpdateNotetype updates an existing notetype in the collection.
func (c *Collection) UpdateNotetype(notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		original, err := getNotetype(tx, notetype.ID)
		if err != nil {
			return err
		}

		if fields := renamedAndRemovedFields(notetype, original); len(fields) > 0 {
			err = updateTemplatesForChangedFields(notetype.Templates, fields)
			if err != nil {
				return err
			}
		}

		err = updateNotesForChangedFields(tx, notetype, len(original.Fields), original.Config.GetSortFieldIdx())
		if err != nil {
			return err
		}

		err = updateCardsForChangedTemplates(tx, notetype, original.Templates)
		if err != nil {
			return err
		}

		notetype.Modified = time.Now()
		notetype.USN = -1

		config, err := proto.Marshal(notetype.Config)
		if err != nil {
			return err
		}

		args := []any{
			notetype.Name,
			timeUnix(notetype.Modified),
			notetype.USN,
			config,
			notetype.ID,
		}
		if err = sqlExecute(tx, updateNotetypeQuery, args...); err != nil {
			return err
		}

		for _, query := range []string{
			deleteFieldsQuery, deleteTemplatesQuery,
		} {
			if err = sqlExecute(tx, query, notetype.ID); err != nil {
				return err
			}
		}

		return addFieldsAndTemplates(tx, notetype)
	})
}

// renamedAndRemovedFields returns a map of field name changes.
// Renamed fields are mapped from old name to new name.
// Removed fields are mapped from old name to an empty string.
func renamedAndRemovedFields(notetype, original *Notetype) map[string]string {
	ords := make(map[int]struct{})
	fields := make(map[string]string)
	for _, f := range notetype.Fields {
		if f.Ordinal == -1 {
			continue
		}
		ords[f.Ordinal] = struct{}{}
		if f.Ordinal < len(original.Fields) {
			existing := original.Fields[f.Ordinal]
			if existing.Name != f.Name {
				fields[existing.Name] = f.Name
			}
		}
	}
	for i, f := range original.Fields {
		if _, ok := ords[i]; !ok {
			fields[f.Name] = ""
		}
	}
	return fields
}

// updateTemplatesForChangedFields handles updates to templates when field names change.
// This function is currently a placeholder and does not perform any operations.
// The user is expected to manually update field references within the templates.
func updateTemplatesForChangedFields(templates []*Template, fields map[string]string) error {
	_ = templates
	_ = fields
	return nil
}

// updateNotesForChangedFields handles updates to notes when the notetype's field
// structure changes (e.g., fields are added, removed, or reordered).
func updateNotesForChangedFields(tx *sql.Tx, notetype *Notetype, previousFieldCount int, previousSortIdx uint32) error {
	ords := sliceMap(notetype.Fields, func(f *Field) int {
		return f.Ordinal
	})
	changed := fieldOrdsChanged(ords, previousFieldCount)
	if changed || notetype.Config.GetSortFieldIdx() != previousSortIdx {
		opts := &ListNotesOptions{
			NotetypeID: &notetype.ID,
		}
		for note, err := range listNotes(tx, opts) {
			if err != nil {
				return err
			}
			if changed {
				reorderNoteFields(note, ords)
			}
			if err = updateNoteWithoutCards(tx, note, notetype); err != nil {
				return err
			}
		}
	}
	return nil
}

// fieldOrdsChanged checks if the order or number of fields has changed.
func fieldOrdsChanged(ords []int, previousLen int) bool {
	if len(ords) != previousLen {
		return true
	}
	for i, ord := range ords {
		if i != ord {
			return true
		}
	}
	return false
}

// reorderNoteFields reorders the fields of a note according to a new ordinal mapping.
func reorderNoteFields(note *Note, ords []int) {
	note.Fields = sliceMap(ords, func(ord int) string {
		if 0 <= ord && ord < len(note.Fields) {
			return note.Fields[ord]
		}
		return ""
	})
}

// updateCardsForChangedTemplates handles card generation, deletion, and updates
// when the notetype's templates are modified.
func updateCardsForChangedTemplates(tx *sql.Tx, notetype *Notetype, ordTemplates []*Template) error {
	ords := sliceMap(notetype.Templates, func(t *Template) int {
		return t.Ordinal
	})
	added, removed, moved := templateOrdChanges(ords, len(ordTemplates))

	// remove any cards where the template was deleted
	if len(removed) > 0 {
		opts := &ListCardsOptions{
			NoteID:   &notetype.ID,
			Ordinals: removed,
		}
		for card, err := range listCards(tx, opts) {
			if err != nil {
				return err
			}
			if err = deleteCardAndAddGrave(tx, card); err != nil {
				return err
			}
		}
	}

	// update ordinals for cards with a repositioned template
	if len(moved) > 0 {
		opts := &ListCardsOptions{
			NoteID:   &notetype.ID,
			Ordinals: slices.Collect(maps.Keys(moved)),
		}
		for card, err := range listCards(tx, opts) {
			if err != nil {
				return err
			}
			card.Modified = time.Now()
			card.Ordinal = moved[card.Ordinal]
			if err = updateCard(tx, card); err != nil {
				return err
			}
		}
	}

	// Generate new cards if templates were added or if the front of any template changed.
	if len(added) > 0 || !equalTemplateFronts(notetype.Templates, ordTemplates) {
		type noteInfo struct {
			deckID int64
			cards  []int
		}
		notes := make(map[int64]*noteInfo)
		opts := &ListCardsOptions{NoteID: &notetype.ID}
		for card, err := range listCards(tx, opts) {
			if err != nil {
				return err
			}
			info, ok := notes[card.NoteID]
			if !ok {
				info = &noteInfo{
					deckID: card.DeckID,
				}
				notes[card.NoteID] = info
			}
			info.cards = append(info.cards, card.Ordinal)
		}

		for id, info := range notes {
			if len(info.cards) == len(notetype.Templates) {
				continue
			}
			note, err := getNote(tx, id)
			if err != nil {
				return err
			}
			for card, err := range generateCards(info.deckID, note, notetype, info.cards) {
				if err != nil {
					return err
				}
				if err = addCard(tx, card); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// templateOrdChanges calculates the changes in template ordinals, identifying
// added, removed, and moved templates.
func templateOrdChanges(ords []int, previousLen int) (added, removed []int, moved map[int]int) {
	for ord := range previousLen {
		removed = append(removed, ord)
	}
	for i, ord := range ords {
		if ord == -1 {
			added = append(added, ord)
		} else {
			if i != ord {
				if moved == nil {
					moved = make(map[int]int)
				}
				moved[ord] = i
			}
			removed = slices.DeleteFunc(removed, func(entry int) bool { return entry == ord })
		}
	}
	return
}

// equalTemplateFronts checks if the question formats (fronts) of two slices of
// templates are equal.
func equalTemplateFronts(tmpls1, tmpls2 []*Template) bool {
	return slices.EqualFunc(tmpls1, tmpls2, func(t1, t2 *Template) bool {
		return t1.Config.QFormat == t2.Config.QFormat
	})
}

// DeleteNotetype deletes a notetype by its ID.
func (c *Collection) DeleteNotetype(id int64) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		if err := sqlExecute(tx, deleteNotetypeQuery, id); err != nil {
			return err
		}
		return deleteNotes(tx, id)
	})
}

// ListNotetypesOptions specifies options for listing notetypes.
type ListNotetypesOptions struct {
	Name *string
}

// ListNotetypes lists all notetypes.
func (c *Collection) ListNotetypes(opts *ListNotetypesOptions) iter.Seq2[*Notetype, error] {
	var args []any
	var conds []string

	if opts != nil {
		if opts.Name != nil {
			name := *opts.Name
			conds = append(conds, "(name = ? OR name LIKE ?)")
			args = append(args, name, name+"+%")
		}
	}

	query := getNotetypeQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	return sqlSelectSeq(c.db, scanNotetype, query, args...)
}

// getNotetype gets a notetype by its ID.
func getNotetype(q sqlQueryer, id int64) (*Notetype, error) {
	return sqlGet(q, scanNotetype, getNotetypeQuery+" WHERE id = ?", id)
}

// addFieldsAndTemplates adds all fields and templates from a notetype struct to the database.
// It sets the ordinal for each field and template based on its slice index.
func addFieldsAndTemplates(tx *sql.Tx, notetype *Notetype) error {
	for i, f := range notetype.Fields {
		f.Ordinal = i
		if err := addField(tx, notetype.ID, f); err != nil {
			return err
		}
	}

	for i, t := range notetype.Templates {
		t.Ordinal = i
		t.Modified = notetype.Modified
		if err := addTemplate(tx, notetype.ID, t); err != nil {
			return err
		}
	}

	return nil
}

// addField adds a field to a notetype.
func addField(tx *sql.Tx, notetypeID int64, field *Field) error {
	if field.Config == nil {
		field.Config = &pb.FieldConfig{}
	}
	config, err := proto.Marshal(field.Config)
	if err != nil {
		return err
	}
	return sqlExecute(tx, addFieldQuery, notetypeID, field.Ordinal, field.Name, config)
}

// listFields lists all fields for a notetype.
func listFields(q sqlQueryer, notetypeID int64) ([]*Field, error) {
	fn := func(_ sqlQueryer, row sqlRow) (*Field, error) {
		var f Field
		var config []byte
		if err := row.Scan(&f.Ordinal, &f.Name, &config); err != nil {
			return nil, err
		}
		f.Config = new(pb.FieldConfig)
		if err := proto.Unmarshal(config, f.Config); err != nil {
			return nil, err
		}
		return &f, nil
	}
	return sqlSelect(q, fn, listFieldsQuery, notetypeID)
}

// addTemplate adds a template to a notetype.
func addTemplate(tx *sql.Tx, notetypeID int64, template *Template) error {
	config, err := proto.Marshal(template.Config)
	if err != nil {
		return err
	}

	args := []any{
		notetypeID,
		template.Ordinal,
		template.Name,
		timeUnix(template.Modified),
		template.USN,
		config,
	}
	return sqlExecute(tx, addTemplateQuery, args...)
}

// listTemplates lists all templates for a notetype.
func listTemplates(q sqlQueryer, notetypeID int64) ([]*Template, error) {
	fn := func(_ sqlQueryer, row sqlRow) (*Template, error) {
		var t Template
		var mod int64
		var config []byte
		if err := row.Scan(&t.Ordinal, &t.Name, &mod, &t.USN, &config); err != nil {
			return nil, err
		}
		t.Modified = time.Unix(mod, 0)
		t.Config = new(pb.TemplateConfig)
		if err := proto.Unmarshal(config, t.Config); err != nil {
			return nil, err
		}
		return &t, nil
	}
	return sqlSelect(q, fn, listTemplatesQuery, notetypeID)
}

// scanNotetype scans a notetype from a database row.
func scanNotetype(q sqlQueryer, row sqlRow) (*Notetype, error) {
	var nt Notetype
	var mod int64
	var config []byte
	err := row.Scan(&nt.ID, &nt.Name, &mod, &nt.USN, &config)
	if err != nil {
		return nil, err
	}

	nt.Modified = time.Unix(mod, 0)
	nt.Config = new(pb.NotetypeConfig)
	if err = proto.Unmarshal(config, nt.Config); err != nil {
		return nil, err
	}

	fields, err := listFields(q, nt.ID)
	if err != nil {
		return nil, err
	}
	nt.Fields = fields

	templates, err := listTemplates(q, nt.ID)
	if err != nil {
		return nil, err
	}
	nt.Templates = templates

	return &nt, nil
}

// addDefaultNotetypes adds the default notetypes to the database.
func addDefaultNotetypes(e sqlExecer) error {
	// TODO add default notetypes
	return nil
}
