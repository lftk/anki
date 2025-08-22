package anki

import (
	"slices"
	"testing"
)

func TestClozeNumberInFields(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		want    []int64
		wantErr bool
	}{
		{
			name:   "no cloze",
			fields: []string{"This is a test.", "Another field."},
			want:   nil,
		},
		{
			name:   "single cloze",
			fields: []string{"This is a {{c1::test}}.", "Another field."},
			want:   []int64{1},
		},
		{
			name:   "multiple clozes same number",
			fields: []string{"This is a {{c1::test}} and another {{c1::example}}.", "Another field."},
			want:   []int64{1},
		},
		{
			name:   "multiple clozes different numbers",
			fields: []string{"This is a {{c1::test}} and another {{c2::example}}.", "Another {{c3::field}}."},
			want:   []int64{1, 2, 3},
		},
		{
			name:   "non-sequential cloze numbers",
			fields: []string{"This is a {{c3::test}} and another {{c1::example}}.", "Another {{c2::field}}."},
			want:   []int64{3, 1, 2},
		},
		{
			name:   "invalid cloze number",
			fields: []string{"This is a {{cX::test}}.", "Another field."},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := clozeNumberInFields(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("clozeNumberInFields(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("clozeNumberInFields(%s) got = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
