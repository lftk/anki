package anki

import (
	"iter"
	"time"
)

type DeckConfig struct {
	ID       int64
	Name     string
	Modified time.Time
	USN      int
	Config   []byte
}

func (c *Collection) GetDeckConfig(id int64) (*DeckConfig, error) {
	config := &DeckConfig{}
	var modSecs int64
	err := c.db.QueryRow("SELECT id, name, mtime_secs, usn, config FROM deck_config WHERE id = ?", id).Scan(
		&config.ID, &config.Name, &modSecs, &config.USN, &config.Config)
	if err != nil {
		return nil, err
	}
	config.Modified = time.Unix(modSecs, 0)
	return config, err
}

func (c *Collection) ListDeckConfigs() iter.Seq2[*DeckConfig, error] {
	return func(yield func(*DeckConfig, error) bool) {
		rows, err := c.db.Query("SELECT id, name, mtime_secs, usn, config FROM deck_config")
		if err != nil {
			yield(nil, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			config := &DeckConfig{}
			var modSecs int64
			if err := rows.Scan(&config.ID, &config.Name, &modSecs, &config.USN, &config.Config); err != nil {
				yield(nil, err)
				return
			}
			config.Modified = time.Unix(modSecs, 0)
			if !yield(config, nil) {
				return
			}
		}
	}
}
