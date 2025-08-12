package anki

import (
	"iter"
)

type Tag struct {
	Name     string
	USN      int64
	Expanded bool
}

func (c *Collection) SetTag(tag *Tag) error {
	const query = `
INSERT
  OR REPLACE INTO tags (tag, usn, collapsed)
VALUES (?, ?, ?)	
`
	return sqlExecute(c.db, query, tag.Name, tag.USN, !tag.Expanded)
}

func (c *Collection) DeleteTag(name string) error {
	const query = `DELETE FROM tags WHERE tag = ?`

	return sqlExecute(c.db, query, name)
}

func (c *Collection) GetTag(name string) (*Tag, error) {
	const query = `SELECT tag, usn, collapsed FROM tags WHERE tag = ?`

	return sqlGet(c.db, scanTag, query, name)
}

func (c *Collection) ListTags() iter.Seq2[*Tag, error] {
	const query = `SELECT tag, usn, collapsed FROM tags`

	return sqlSelectSeq(c.db, scanTag, query)
}

func scanTag(_ sqlQueryer, row sqlRow) (*Tag, error) {
	var name string
	var usn int64
	var collapsed bool
	if err := row.Scan(&name, &usn, &collapsed); err != nil {
		return nil, err
	}
	return &Tag{
		Name:     name,
		USN:      usn,
		Expanded: !collapsed,
	}, nil
}
