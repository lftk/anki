package anki

import (
	_ "embed"
)

//go:embed queries/add_card.sql
var addCardQuery string

//go:embed queries/get_card.sql
var getCardQuery string

//go:embed queries/add_note.sql
var addNoteQuery string

//go:embed queries/get_deck.sql
var getDeckQuery string

//go:embed queries/get_note.sql
var getNoteQuery string

//go:embed queries/get_notetype.sql
var getNotetypeQuery string

//go:embed queries/get_tag.sql
var getTagQuery string
