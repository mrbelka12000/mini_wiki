package mini_wiki

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	_ "github.com/lib/pq"
)

const maxContentSize = 1048575

type repository struct {
	db *sql.DB
}

func newRepository(db *sql.DB) *repository {
	return &repository{db: db}
}

func DatabaseConnect(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.PGURL)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return db, nil
}

func (r *repository) Insert(ctx context.Context, content io.Reader, title, objectName string) error {

	chank := make([]byte, maxContentSize/2)

	for {
		n, err := content.Read(chank)
		if err != nil {
			if err != io.EOF {
				return fmt.Errorf("read: %w", err)
			}
		}

		text := string(chank[:n])
		if text == "" {
			break
		}
		text = cleanString(text) + title

		_, err = r.db.ExecContext(ctx, `
	INSERT INTO files (
	title, text, file_key, search_vector) 
	VALUES ($1, $2, $3, strip(to_tsvector('simple', regexp_replace($4, '[^\u0000-\u007F]+', ' ', 'g')

)))
`, title, text, objectName, text)
		fmt.Println(text)
		if err != nil {
			return fmt.Errorf("insert: %w", err)
		}

		if n == 0 {
			break
		}
	}

	return nil
}

func (r *repository) Find(ctx context.Context, toFind string) ([]string, error) {
	query := `
SELECT DISTINCT file_key
FROM files
WHERE search_vector @@ to_tsquery('simple', $1)
;`

	var objectNames []string
	rows, err := r.db.QueryContext(ctx, query, toFind)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		objectNames = append(objectNames, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return objectNames, nil
}

func (r *repository) Delete(ctx context.Context, objectName string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE file_key = $1`, objectName)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

func (r *repository) IncrementFileNameVersion(ctx context.Context, objectName string) error {
	_, err := r.db.ExecContext(ctx, `
insert into file_names(
                       file_key, count
) VALUES (
          $1, 1
         ) on conflict(file_key) DO UPDATE set count = file_names.count+1 where file_names.file_key = $1;
`, objectName)
	if err != nil {
		return fmt.Errorf("increment: %w", err)
	}

	return nil
}

func (r *repository) GetFileNamesVersion(ctx context.Context, objectName string) (version int, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT count from file_names where file_key = $1`, objectName).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	return version, err
}

func cleanString(input string) string {
	var cleanBuilder strings.Builder
	for _, r := range input {
		if r == utf8.RuneError {
			continue // Skip invalid runes
		}
		if r < 32 && r != 10 && r != 13 { // Remove control characters except newline and carriage return
			continue
		}
		cleanBuilder.WriteRune(r)
	}
	return cleanBuilder.String()
}
