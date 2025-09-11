package anki

import (
	"regexp"
	"strconv"
)

// clozeNumberRe is a regular expression to find cloze numbers in a string.
var clozeNumberRe = regexp.MustCompile(`\{\{c(\d+)::`)

// clozeNumberInFields extracts all unique cloze numbers from a slice of strings (fields).
func clozeNumberInFields(fields []string) ([]int, error) {
	seen := make(map[int]struct{})
	ords := make([]int, 0, len(fields))
	for _, field := range fields {
		matches := clozeNumberRe.FindAllStringSubmatch(field, -1)
		for _, match := range matches {
			if len(match) > 1 {
				i, err := strconv.ParseInt(match[1], 10, 32)
				if err != nil {
					return nil, err
				}
				if _, ok := seen[int(i)]; !ok {
					seen[int(i)] = struct{}{}
					ords = append(ords, int(i))
				}
			}
		}
	}
	return ords, nil
}
