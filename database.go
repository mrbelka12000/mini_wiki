package mini_wiki

import (
	"context"
	"database/sql"
	"fmt"
	"io"

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
		text += title
		_, err = r.db.ExecContext(ctx, `
	INSERT INTO files (title, text, file_key, search_vector) VALUES ($1, $2, $3, strip(to_tsvector('simple',$4)))
`, title, text, objectName, text)
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
