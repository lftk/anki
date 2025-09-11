package anki

// addCardGrave adds a grave entry for a deleted card.
// The usn and type are set to 0, as is standard for card deletions.
func addCardGrave(e sqlExecer, cardID int64) error {
	return sqlExecute(e, addGraveQuery, 0, cardID, 0)
}
