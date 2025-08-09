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
	const query = `SELECT id, name, mtime_secs, usn, config FROM notetypes WHERE id = ?`

	var nt Notetype
	var mod int64
	var config []byte
	err := c.db.QueryRow(query, id).Scan(&nt.ID, &nt.Name, &mod, &nt.USN, config)
	if err != nil {
		return nil, err
	}

	nt.Modified = time.Unix(mod, 0)
	nt.Config = new(NotetypeConfig)
	if err = proto.Unmarshal(config, nt.Config); err != nil {
		return nil, err
	}

	fields, err := c.listFields(nt.ID)
	if err != nil {
		return nil, err
	}
	nt.Fields = fields

	templates, err := c.listTemplates(nt.ID)
	if err != nil {
		return nil, err
	}
	nt.Templates = templates

	return &nt, nil
}

func (c *Collection) AddNotetype(notetype *Notetype) error {
	const query = `INSERT INTO notetypes (id, name, mtime_secs, usn, config) VALUES (?, ?, ?, ?, ?)`

	return withTransaction(c.db, func(tx *sql.Tx) error {
		notetype.Modified = time.Now()
		notetype.USN = -1

		config, err := proto.Marshal(notetype.Config)
		if err != nil {
			return err
		}

		_, err = tx.Exec(query, notetype.ID, notetype.Name, notetype.Modified.Unix(), notetype.USN, config)
		if err != nil {
			return err
		}

		for _, f := range notetype.Fields {
			if err := c.addField(tx, notetype.ID, f); err != nil {
				return err
			}
		}

		for _, t := range notetype.Templates {
			t.Modified = notetype.Modified
			if err := c.addTemplate(tx, notetype.ID, t); err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Collection) UpdateNotetype(notetype *Notetype) error {
	const query = `UPDATE notetypes SET name = ?, mtime_secs = ?, usn = ?, config = ? WHERE id = ?`

	return withTransaction(c.db, func(tx *sql.Tx) error {
		notetype.Modified = time.Now()
		notetype.USN = -1

		config, err := proto.Marshal(notetype.Config)
		if err != nil {
			return err
		}

		_, err = tx.Exec(query, notetype.Name, notetype.Modified.Unix(), notetype.USN, config, notetype.ID)
		if err != nil {
			return err
		}

		for _, query := range []string{
			`DELETE FROM fields WHERE ntid = ?`,
			`DELETE FROM templates WHERE ntid = ?`,
		} {
			if _, err = tx.Exec(query, notetype.ID); err != nil {
				return err
			}
		}

		for _, f := range notetype.Fields {
			if err = c.addField(tx, notetype.ID, f); err != nil {
				return err
			}
		}

		for _, t := range notetype.Templates {
			t.Modified = notetype.Modified
			if err = c.addTemplate(tx, notetype.ID, t); err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Collection) DeleteNotetype(id int64) error {
	return withTransaction(c.db, func(tx *sql.Tx) error {
		for _, query := range []string{
			`DELETE FROM notetypes WHERE id = ?`,
			`DELETE FROM fields WHERE ntid = ?`,
			`DELETE FROM templates WHERE ntid = ?`,
		} {
			if _, err := tx.Exec(query, id); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Collection) ListNotetypes() iter.Seq2[*Notetype, error] {
	const query = `SELECT id, name, mtime_secs, usn, config FROM notetypes`

	return func(yield func(*Notetype, error) bool) {
		rows, err := c.db.Query(query)
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var nt Notetype
			var mod int64
			var config []byte
			if err = rows.Scan(&nt.ID, &nt.Name, &mod, &nt.USN, &config); err != nil {
				yield(nil, err)
				return
			}

			nt.Modified = time.Unix(mod, 0)
			nt.Config = new(NotetypeConfig)
			if err = proto.Unmarshal(config, nt.Config); err != nil {
				yield(nil, err)
				return
			}

			fields, err := c.listFields(nt.ID)
			if err != nil {
				yield(nil, err)
				return
			}
			nt.Fields = fields

			templates, err := c.listTemplates(nt.ID)
			if err != nil {
				yield(nil, err)
				return
			}
			nt.Templates = templates

			if !yield(&nt, nil) {
				return
			}
		}
	}
}

func (c *Collection) listFields(notetypeID int64) ([]*Field, error) {
	const query = `SELECT ord, name, config FROM fields WHERE ntid = ? ORDER BY ord`

	rows, err := c.db.Query(query, notetypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []*Field
	for rows.Next() {
		var f Field
		var config []byte
		if err = rows.Scan(&f.Ordinal, &f.Name, &config); err != nil {
			return nil, err
		}

		f.Config = new(FieldConfig)
		if err = proto.Unmarshal(config, f.Config); err != nil {
			return nil, err
		}

		fields = append(fields, &f)
	}
	return fields, nil
}

func (c *Collection) addField(tx *sql.Tx, notetypeID int64, field *Field) error {
	const query = `INSERT INTO fields (ntid, ord, name, config) VALUES (?, ?, ?, ?)`

	config, err := proto.Marshal(field.Config)
	if err != nil {
		return err
	}

	_, err = tx.Exec(query, notetypeID, field.Ordinal, field.Name, config)
	return err
}

func (c *Collection) listTemplates(notetypeID int64) ([]*Template, error) {
	const query = `SELECT ord, name, mtime_secs, usn, config FROM templates WHERE ntid = ? ORDER BY ord`

	rows, err := c.db.Query(query, notetypeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*Template
	for rows.Next() {
		var t Template
		var mod int64
		var config []byte
		if err = rows.Scan(&t.Ordinal, &t.Name, &mod, &t.USN, &config); err != nil {
			return nil, err
		}

		t.Modified = time.Unix(mod, 0)
		t.Config = new(TemplateConfig)
		if err = proto.Unmarshal(config, t.Config); err != nil {
			return nil, err
		}

		templates = append(templates, &t)
	}
	return templates, nil
}

func (c *Collection) addTemplate(tx *sql.Tx, notetypeID int64, template *Template) error {
	const query = `INSERT INTO templates (ntid, ord, name, mtime_secs, usn, config) VALUES (?, ?, ?, ?, ?, ?)`

	config, err := proto.Marshal(template.Config)
	if err != nil {
		return err
	}

	_, err = tx.Exec(query, notetypeID, template.Ordinal, template.Name, template.Modified.Unix(), template.USN, config)
	return err
}
