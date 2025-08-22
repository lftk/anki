package anki

import (
	_ "embed"
)

//go:embed queries/add_card.sql
var addCardQuery string

//go:embed queries/add_note.sql
var addNoteQuery string
