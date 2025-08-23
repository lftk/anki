package anki

import (
	"regexp"
	"strconv"
)

// clozeNumberRe is a regular expression to find cloze numbers in a string.
var clozeNumberRe = regexp.MustCompile(`\{\{c(\d+)::`)

// clozeNumberInFields extracts all unique cloze numbers from a slice of strings (fields).
func clozeNumberInFields(fields []string) ([]int64, error) {
	seen := make(map[int64]struct{})
	ords := make([]int64, 0, len(fields))
	for _, field := range fields {
		matches := clozeNumberRe.FindAllStringSubmatch(field, -1)
		for _, match := range matches {
			if len(match) > 1 {
				i, err := strconv.ParseInt(match[1], 10, 64)
				if err != nil {
					return nil, err
				}
				if _, ok := seen[i]; !ok {
					seen[i] = struct{}{}
					ords = append(ords, i)
				}
			}
		}
	}
	return ords, nil
}
