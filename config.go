package anki

import (
	"strconv"
	"time"
)

func (c *Collection) GetConfig(key string) ([]byte, error) {
	var val []byte
	err := c.db.QueryRow("SELECT val FROM config WHERE key = ?", key).Scan(&val)
	return val, err
}

func (c *Collection) SetConfig(key string, value []byte) error {
	_, err := c.db.Exec("REPLACE INTO config (key, usn, mtime_secs, val) VALUES (?, ?, ?, ?)",
		key, -1, time.Now().Unix(), value)
	return err
}

func (c *Collection) RemoveConfig(key string) error {
	_, err := c.db.Exec("DELETE FROM config WHERE key = ?", key)
	return err
}

func (c *Collection) GetConfigString(key string) (string, error) {
	val, err := c.GetConfig(key)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (c *Collection) SetConfigString(key, value string) error {
	return c.SetConfig(key, []byte(value))
}

func (c *Collection) GetConfigBool(key string) (bool, error) {
	val, err := c.GetConfig(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(string(val))
}

func (c *Collection) SetConfigBool(key string, value bool) error {
	return c.SetConfig(key, []byte(strconv.FormatBool(value)))
}
