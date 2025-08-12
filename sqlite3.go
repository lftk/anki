package anki

import (
	"database/sql"
	"strings"
	"unicode"

	"github.com/mattn/go-sqlite3"
)

func init() {
	sql.Register("sqlite3_ext", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterCollation("unicase", unicase)
		},
	})
}

func unicase(s1, s2 string) int {
	return strings.Compare(
		strings.Map(unicode.ToLower, s1),
		strings.Map(unicode.ToLower, s2),
	)
}

func sqlite3Open(dataSourceName string) (*sql.DB, error) {
	return sql.Open("sqlite3_ext", dataSourceName)
}
