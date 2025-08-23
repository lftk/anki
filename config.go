package anki

import (
	"iter"
	"time"
)

type Config struct {
	Key      string
	Value    []byte
	USN      int64
	Modified time.Time
}

func (c *Collection) SetConfig(config *Config) error {
	const query = `
INSERT
  OR REPLACE INTO config (key, usn, mtime_secs, val)
VALUES (?, ?, ?, ?)	
`
	return sqlExecute(c.db, query, config.Key, config.USN, config.Modified.Unix(), config.Value)
}

func (c *Collection) GetConfig(key string) (*Config, error) {
	const query = `SELECT key, usn, mtime_secs, val FROM config WHERE key = ?`

	return sqlGet(c.db, scanConfig, query, key)
}

func (c *Collection) DeleteConfig(key string) error {
	return sqlExecute(c.db, "DELETE FROM config WHERE key = ?", key)
}

type ListConfigsOptions struct{}

func (c *Collection) ListConfigs(*ListConfigsOptions) iter.Seq2[*Config, error] {
	const query = `SELECT key, usn, mtime_secs, val FROM config`

	return sqlSelectSeq(c.db, scanConfig, query)
}

func scanConfig(_ sqlQueryer, row sqlRow) (*Config, error) {
	var c Config
	var mod int64
	if err := row.Scan(&c.Key, &c.USN, &mod, &c.Value); err != nil {
		return nil, err
	}
	c.Modified = time.Unix(mod, 0)
	return &c, nil
}
