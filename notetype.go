package anki

import (
	"database/sql"
	"iter"
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

// SimpleNotetypeConfig creates a simple notetype configuration with the given CSS.
func SimpleNotetypeConfig(css string) *pb.NotetypeConfig {
	return &pb.NotetypeConfig{Css: css}
}

// Field represents a field in a notetype.
type Field struct {
	Ordinal int
	Name    string
	Config  *pb.FieldConfig
}

// Template represents a template in a notetype.
type Template struct {
	Ordinal  int
	Name     string
	Modified time.Time
	USN      int64
	Config   *pb.TemplateConfig
}

// SimpleTemplateConfig creates a simple template configuration with the given
// front and back formats.
func SimpleTemplateConfig(front, back string) *pb.TemplateConfig {
	return &pb.TemplateConfig{
		QFormat: front,
		AFormat: back,
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

		for _, f := range notetype.Fields {
			if err := addField(tx, notetype.ID, f); err != nil {
				return err
			}
		}

		for _, t := range notetype.Templates {
			t.Modified = notetype.Modified
			if err := addTemplate(tx, notetype.ID, t); err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdateNotetype updates an existing notetype in the collection.
func (c *Collection) UpdateNotetype(notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
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

		if err = sqlExecute(tx, deleteFieldsQuery, notetype.ID); err != nil {
			return err
		}
		if err = sqlExecute(tx, deleteTemplatesQuery, notetype.ID); err != nil {
			return err
		}

		for _, f := range notetype.Fields {
			if err = addField(tx, notetype.ID, f); err != nil {
				return err
			}
		}

		for _, t := range notetype.Templates {
			t.Modified = notetype.Modified
			if err = addTemplate(tx, notetype.ID, t); err != nil {
				return err
			}
		}

		return nil
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
type ListNotetypesOptions struct{}

// ListNotetypes lists all notetypes.
func (c *Collection) ListNotetypes(opts *ListNotetypesOptions) iter.Seq2[*Notetype, error] {
	return sqlSelectSeq(c.db, scanNotetype, getNotetypeQuery)
}

// getNotetype gets a notetype by its ID.
func getNotetype(q sqlQueryer, id int64) (*Notetype, error) {
	return sqlGet(q, scanNotetype, getNotetypeQuery+" WHERE id = ?", id)
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
