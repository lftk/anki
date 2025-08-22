package anki

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/alexkappa/mustache"
)

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

// fieldRequirements parses an Anki template and returns a matcher for field requirements
// based on the provided field ordinals mapping. The matcher can be used to determine
// which fields are actually used in the template.
//
// The implementation is based on the Rust version in Anki's codebase:
// see https://github.com/ankitects/anki/blob/72abb7ec5bfdb0e00029f721fc5f5cf13b5085a4/rslib/src/template.rs#L704-L734.
func fieldRequirements(template string, fieldOrdinals map[string]int) (fieldRequirementsMatcher, error) {
	// Create a SHA256 hash to generate a unique sentinel value
	h := sha256.New()

	// Parse the template while simultaneously hashing its content
	// io.TeeReader allows reading from the template while writing to the hash
	t, err := mustache.Parse(
		io.TeeReader(
			strings.NewReader(template), h,
		),
	)
	if err != nil {
		return nil, err
	}

	// Generate a unique sentinel value from the template's hash
	// This will be used to detect if fields are actually used in the template
	sentinel := hex.EncodeToString(h.Sum(nil))

	var ordinals []int
	// Check each field to see if it's used in the template
	for field, ordinal := range fieldOrdinals {
		// Create a test context with just this field set to our sentinel value
		fieldValues := map[string]string{
			field: sentinel,
		}

		// Render the template with just this field set
		if rendered, err := t.RenderString(fieldValues); err != nil {
			return nil, err
		} else if strings.Contains(rendered, sentinel) {
			// If the sentinel appears in the output, this field is used
			ordinals = append(ordinals, ordinal)
		}
	}

	// If we found any required fields, return a matcher that checks for any of them
	if len(ordinals) > 0 {
		return fieldRequirementsMatchAny(ordinals), nil
	}

	// If no fields were found to be required individually, check if all fields together
	// are required (this would catch cases where fields are only used in conditionals)
	fieldValues := make(map[string]string)
	for field := range fieldOrdinals {
		fieldValues[field] = sentinel
	}

	// Collect all field ordinals initially
	ordinals = slices.Collect(maps.Values(fieldOrdinals))

	// Check if we can remove each field and still have the sentinel appear in the output
	for field, ordinal := range fieldOrdinals {
		// can we remove this field and still render?
		delete(fieldValues, field)
		if rendered, err := t.RenderString(fieldValues); err != nil {
			return nil, err
		} else if strings.Contains(rendered, sentinel) {
			// If sentinel still appears, this field isn't strictly required
			ordinals = slices.DeleteFunc(ordinals, func(item int) bool {
				return ordinal == item
			})
		}
		fieldValues[field] = sentinel
	}

	// If we have remaining ordinals after filtering, check if they're all required together
	if len(ordinals) != 0 {
		if rendered, err := t.RenderString(fieldValues); err != nil {
			return nil, err
		} else if strings.Contains(rendered, sentinel) {
			// If sentinel appears with all fields set, these fields are required together
			return fieldRequirementsMatchAll(ordinals), nil
		}
	}

	// If no fields are required, return the none matcher
	return fieldRequirementsMatchNone, nil
}

// fieldRequirementsMatcher defines the interface for checking field requirements
type fieldRequirementsMatcher interface {
	MatchFieldRequirements(fieldValues []string) bool
}

// fieldRequirementsMatcherFunc is a function type that implements the matcher interface
type fieldRequirementsMatcherFunc func(fieldValues []string) bool

func (f fieldRequirementsMatcherFunc) MatchFieldRequirements(fieldValues []string) bool {
	return f(fieldValues)
}

// fieldRequirementsMatchNone is a matcher that always returns false (no fields required)
var fieldRequirementsMatchNone = fieldRequirementsMatcherFunc(func(_ []string) bool { return false })

// fieldRequirementsMatchAny creates a matcher that checks if any of the specified fields are present
func fieldRequirementsMatchAny(fieldOrdinals []int) fieldRequirementsMatcherFunc {
	return func(fieldValues []string) bool {
		for _, ord := range fieldOrdinals {
			if ord < len(fieldValues) && fieldValues[ord] != "" {
				return true
			}
		}
		return false
	}
}

// fieldRequirementsMatchAll creates a matcher that checks if all specified fields are present
func fieldRequirementsMatchAll(fieldOrdinals []int) fieldRequirementsMatcherFunc {
	return func(fieldValues []string) bool {
		for _, ord := range fieldOrdinals {
			if ord >= len(fieldValues) || fieldValues[ord] == "" {
				return false
			}
		}
		return true
	}
}
