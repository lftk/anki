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
	return sqlExecute(c.db, setTagQuery, tag.Name, tag.USN, !tag.Expanded)
}

func (c *Collection) DeleteTag(name string) error {
	return sqlExecute(c.db, deleteTagQuery, name)
}

func (c *Collection) GetTag(name string) (*Tag, error) {
	return sqlGet(c.db, scanTag, getTagQuery+" WHERE tag = ?", name)
}

type ListTagsOptions struct{}

func (c *Collection) ListTags(opts *ListTagsOptions) iter.Seq2[*Tag, error] {
	return sqlSelectSeq(c.db, scanTag, getTagQuery)
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
