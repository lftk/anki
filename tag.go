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
	return func(yield func(*Tag, error) bool) {
		rows, err := c.db.Query("SELECT tag, usn, collapsed, config FROM tags")
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			tag := &Tag{}
			if err := rows.Scan(&tag.Name, &tag.USN, &tag.Collapsed, &tag.Config); err != nil {
				yield(nil, err)
				return
			}
			if !yield(tag, nil) {
				return
			}
		}
	}
}
