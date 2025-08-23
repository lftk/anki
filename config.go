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
	args := []any{
		config.Key,
		config.USN,
		timeUnix(config.Modified),
		config.Value,
	}
	return sqlExecute(c.db, setConfigQuery, args...)
}

func (c *Collection) GetConfig(key string) (*Config, error) {
	return sqlGet(c.db, scanConfig, getConfigQuery+" WHERE key = ?", key)
}

func (c *Collection) DeleteConfig(key string) error {
	return sqlExecute(c.db, deleteConfigQuery, key)
}

type ListConfigsOptions struct{}

func (c *Collection) ListConfigs(*ListConfigsOptions) iter.Seq2[*Config, error] {
	return sqlSelectSeq(c.db, scanConfig, getConfigQuery)
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
