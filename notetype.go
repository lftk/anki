package anki

import (
	"database/sql"
	"iter"
	"time"

	"google.golang.org/protobuf/proto"
)

type Notetype struct {
	ID        int64
	Name      string
	Modified  time.Time
	USN       int64
	Fields    []*Field
	Templates []*Template
	Config    *NotetypeConfig
}

type Field struct {
	Ordinal int
	Name    string
	Config  *FieldConfig
}

type Template struct {
	Ordinal  int
	Name     string
	Modified time.Time
	USN      int64
	Config   *TemplateConfig
}

func (c *Collection) GetNotetype(id int64) (*Notetype, error) {
	return sqlGet(c.db, scanNotetype, getNotetypeQuery+" WHERE id = ?", id)
}

func (c *Collection) AddNotetype(notetype *Notetype) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		notetype.Modified = time.Now()
		notetype.USN = -1

		config, err := proto.Marshal(notetype.Config)
		if err != nil {
			return err
		}

		args := []any{
			notetype.ID,
			notetype.Name,
			notetype.Modified.Unix(),
			notetype.USN,
			config,
		}
		if err = sqlExecute(tx, addNotetypeQuery, args...); err != nil {
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
			notetype.Modified.Unix(),
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

func (c *Collection) DeleteNotetype(id int64) error {
	return sqlTransact(c.db, func(tx *sql.Tx) error {
		if err := sqlExecute(tx, deleteNotetypeQuery, id); err != nil {
			return err
		}
		return deleteNotes(tx, id)
	})
}

type ListNotetypesOptions struct{}

func (c *Collection) ListNotetypes(opts *ListNotetypesOptions) iter.Seq2[*Notetype, error] {
	return sqlSelectSeq(c.db, scanNotetype, getNotetypeQuery)
}

func addField(tx *sql.Tx, notetypeID int64, field *Field) error {
	config, err := proto.Marshal(field.Config)
	if err != nil {
		return err
	}
	return sqlExecute(tx, addFieldQuery, notetypeID, field.Ordinal, field.Name, config)
}

func listFields(q sqlQueryer, notetypeID int64) ([]*Field, error) {
	fn := func(_ sqlQueryer, row sqlRow) (*Field, error) {
		var f Field
		var config []byte
		if err := row.Scan(&f.Ordinal, &f.Name, &config); err != nil {
			return nil, err
		}
		f.Config = new(FieldConfig)
		if err := proto.Unmarshal(config, f.Config); err != nil {
			return nil, err
		}
		return &f, nil
	}
	return sqlSelect(q, fn, listFieldsQuery, notetypeID)
}

func addTemplate(tx *sql.Tx, notetypeID int64, template *Template) error {
	config, err := proto.Marshal(template.Config)
	if err != nil {
		return err
	}

	args := []any{
		notetypeID,
		template.Ordinal,
		template.Name,
		template.Modified.Unix(),
		template.USN,
		config,
	}
	return sqlExecute(tx, addTemplateQuery, args...)
}

func listTemplates(q sqlQueryer, notetypeID int64) ([]*Template, error) {
	fn := func(_ sqlQueryer, row sqlRow) (*Template, error) {
		var t Template
		var mod int64
		var config []byte
		if err := row.Scan(&t.Ordinal, &t.Name, &mod, &t.USN, &config); err != nil {
			return nil, err
		}
		t.Modified = time.Unix(mod, 0)
		t.Config = new(TemplateConfig)
		if err := proto.Unmarshal(config, t.Config); err != nil {
			return nil, err
		}
		return &t, nil
	}
	return sqlSelect(q, fn, listTemplatesQuery, notetypeID)
}

func scanNotetype(q sqlQueryer, row sqlRow) (*Notetype, error) {
	var nt Notetype
	var mod int64
	var config []byte
	err := row.Scan(&nt.ID, &nt.Name, &mod, &nt.USN, &config)
	if err != nil {
		return nil, err
	}

	nt.Modified = time.Unix(mod, 0)
	nt.Config = new(NotetypeConfig)
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
