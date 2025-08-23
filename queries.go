package anki

import (
	_ "embed"
)

//go:embed queries/schema.sql
var schemaQuery string

//go:embed queries/add_card.sql
var addCardQuery string

//go:embed queries/add_deck.sql
var addDeckQuery string

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

//go:embed queries/delete_cards.sql
var deleteCardsQuery string

//go:embed queries/list_note_ids.sql
var listNoteIDsQuery string

//go:embed queries/delete_note.sql
var deleteNoteQuery string

//go:embed queries/update_note.sql
var updateNoteQuery string

//go:embed queries/add_notetype.sql
var addNotetypeQuery string

//go:embed queries/update_notetype.sql
var updateNotetypeQuery string

//go:embed queries/delete_fields.sql
var deleteFieldsQuery string

//go:embed queries/delete_templates.sql
var deleteTemplatesQuery string

//go:embed queries/delete_notetype.sql
var deleteNotetypeQuery string

//go:embed queries/add_field.sql
var addFieldQuery string

//go:embed queries/list_fields.sql
var listFieldsQuery string

//go:embed queries/add_template.sql
var addTemplateQuery string

//go:embed queries/list_templates.sql
var listTemplatesQuery string

//go:embed queries/set_tag.sql
var setTagQuery string

//go:embed queries/delete_tag.sql
var deleteTagQuery string
