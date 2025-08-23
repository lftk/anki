package anki

import (
	"iter"
)

// Tag represents a tag in Anki.
type Tag struct {
	Name     string
	USN      int64
	Expanded bool
}

// SetTag adds or updates a tag.
func (c *Collection) SetTag(tag *Tag) error {
	return sqlExecute(c.db, setTagQuery, tag.Name, tag.USN, !tag.Expanded)
}

// DeleteTag deletes a tag by name.
func (c *Collection) DeleteTag(name string) error {
	return sqlExecute(c.db, deleteTagQuery, name)
}

// GetTag gets a tag by name.
func (c *Collection) GetTag(name string) (*Tag, error) {
	return sqlGet(c.db, scanTag, getTagQuery+" WHERE tag = ?", name)
}

// ListTagsOptions specifies options for listing tags.
type ListTagsOptions struct{}

// ListTags lists all tags.
func (c *Collection) ListTags(*ListTagsOptions) iter.Seq2[*Tag, error] {
	return sqlSelectSeq(c.db, scanTag, getTagQuery)
}

// scanTag scans a tag from a database row.
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
