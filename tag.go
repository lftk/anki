package anki

import (
	"iter"
)

type Tag struct {
	Name      string
	USN       int
	Collapsed bool
	Config    []byte
}

func (c *Collection) ListTags() iter.Seq2[*Tag, error] {
	const query = `SELECT tag, usn, collapsed, config FROM tags`

	return sqlSelectSeq(c.db, scanTag, query)
}

func scanTag(_ sqlQueryer, row sqlRow) (*Tag, error) {
	var tag Tag
	if err := row.Scan(&tag.Name, &tag.USN, &tag.Collapsed, &tag.Config); err != nil {
		return nil, err
	}
	return &tag, nil
}
