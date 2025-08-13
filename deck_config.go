package anki

import (
	"iter"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/lftk/anki/internal/pb"
)

type DeckConfig struct {
	ID       int64
	Name     string
	Modified time.Time
	USN      int
	Config   *pb.DeckConfig
}

func (c *Collection) SetDeckConfig(config *DeckConfig) error {
	const query = `
INSERT
  OR REPLACE INTO deck_config (id, name, usn, mtime_secs, config)
VALUES (?, ?, ?, ?, ?)	
`

	inner, err := proto.Marshal(config.Config)
	if err != nil {
		return err
	}
	return sqlExecute(c.db, query, config.ID, config.Name, config.USN, config.Modified.Unix(), inner)
}

func (c *Collection) GetDeckConfig(id int64) (*DeckConfig, error) {
	const query = `SELECT id, name, mtime_secs, usn, config FROM deck_config WHERE id = ?`

	return sqlGet(c.db, scanDeckConfig, query, id)
}

func (c *Collection) DeleteDeckConfig(id int64) error {
	return sqlExecute(c.db, "DELETE FROM deck_config WHERE id = ?", id)
}

func (c *Collection) ListDeckConfigs() iter.Seq2[*DeckConfig, error] {
	const query = `SELECT id, name, mtime_secs, usn, config FROM deck_config`

	return sqlSelectSeq(c.db, scanDeckConfig, query)
}

func scanDeckConfig(_ sqlQueryer, row sqlRow) (*DeckConfig, error) {
	var c DeckConfig
	var mod int64
	var config []byte
	if err := row.Scan(&c.ID, &c.Name, &mod, &c.USN, &config); err != nil {
		return nil, err
	}
	c.Modified = time.Unix(mod, 0)
	c.Config = new(pb.DeckConfig)
	if err := proto.Unmarshal(config, c.Config); err != nil {
		return nil, err
	}
	return &c, nil
}
