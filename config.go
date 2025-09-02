package anki

import (
	"encoding/json"
	"iter"
	"time"
)

// Config represents a configuration entry in Anki.
type Config struct {
	Key      string
	Value    []byte
	USN      int64
	Modified time.Time
}

// SetConfig sets a configuration entry.
func (c *Collection) SetConfig(config *Config) error {
	return setConfig(c.db, config)
}

// setConfig sets a configuration entry.
func setConfig(e sqlExecer, config *Config) error {
	args := []any{
		config.Key,
		config.USN,
		timeUnix(config.Modified),
		config.Value,
	}
	return sqlExecute(e, setConfigQuery, args...)
}

// GetConfig gets a configuration entry by key.
func (c *Collection) GetConfig(key string) (*Config, error) {
	return sqlGet(c.db, scanConfig, getConfigQuery+" WHERE key = ?", key)
}

// DeleteConfig deletes a configuration entry by key.
func (c *Collection) DeleteConfig(key string) error {
	return sqlExecute(c.db, deleteConfigQuery, key)
}

// ListConfigsOptions specifies options for listing configuration entries.
type ListConfigsOptions struct{}

// ListConfigs lists all configuration entries.
func (c *Collection) ListConfigs(*ListConfigsOptions) iter.Seq2[*Config, error] {
	return sqlSelectSeq(c.db, scanConfig, getConfigQuery)
}

// scanConfig scans a configuration entry from a database row.
func scanConfig(_ sqlQueryer, row sqlRow) (*Config, error) {
	var c Config
	var mod int64
	if err := row.Scan(&c.Key, &c.USN, &mod, &c.Value); err != nil {
		return nil, err
	}
	c.Modified = time.Unix(mod, 0)
	return &c, nil
}

// initDefaultConfigs initializes default configuration entries.
func initDefaultConfigs(e sqlExecer) error {
	for key, value := range map[string]any{
		"activeDecks":    []int64{1},
		"curDeck":        int64(1),
		"newSpread":      int64(0),
		"collapseTime":   int64(1200),
		"timeLim":        int64(0),
		"estTimes":       true,
		"dueCounts":      true,
		"curModel":       nil,
		"nextPos":        int64(1),
		"sortType":       "noteFld",
		"sortBackwards":  false,
		"addToCur":       true,
		"dayLearnFirst":  false,
		"schedVer":       int64(2),
		"creationOffset": int64(0),
		"sched2021":      true,
	} {
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		config := &Config{
			Key:      key,
			Value:    b,
			USN:      0,
			Modified: timeZero(),
		}
		if err = setConfig(e, config); err != nil {
			return err
		}
	}
	return nil
}
