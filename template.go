package anki

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	"github.com/alexkappa/mustache"
)

// rendersTemplate checks if a template renders to a non-empty card given a set of fields.
func rendersTemplate(template *Template, fields []string) (bool, error) {
	if template == nil || template.Config == nil || template.Config.QFormat == "" {
		// No template or empty QFormat means it can't render anything
		return false, nil
	}

	if len(fields) == 0 {
		// No fields provided, can't render anything
		return false, nil
	}

	h := sha256.New()
	t, err := mustache.Parse(
		io.TeeReader(
			strings.NewReader(template.Config.QFormat), h,
		),
	)
	if err != nil {
		return false, err
	}

	sentinel := hex.EncodeToString(h.Sum(nil))

	fieldValues := make(map[string]string)
	for _, field := range fields {
		fieldValues[field] = sentinel
	}

	output, err := t.RenderString(fieldValues)
	if err != nil {
		return false, err
	}

	// If the sentinel appears in the output, it means the template rendered
	// with the provided fields.
	return strings.Contains(output, sentinel), nil
}
