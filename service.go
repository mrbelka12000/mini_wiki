package mini_wiki

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strings"
)

type (
	Service struct {
		repo    *repository
		log     *slog.Logger
		storage storage
	}

	storage interface {
		UploadFile(ctx context.Context, file io.Reader, objectName, contentType string, fileSize int64) (string, error)
		DownloadFile(ctx context.Context, w http.ResponseWriter, objectName string) error
	}
)

func RunService(db *sql.DB, storage storage, log *slog.Logger, mux *http.ServeMux, cfg Config) error {
	s := &Service{
		repo:    newRepository(db),
		storage: storage,
		log:     log,
	}

	initHandlers(s, mux)

	return http.ListenAndServe(":"+cfg.HTTPPort, mux)
}

func initHandlers(s *Service, mux *http.ServeMux) {
	mux.HandleFunc("/", makeIndexHandler())
	mux.HandleFunc("/upload", makeUploadDataHandler(s))
	mux.HandleFunc("/search", makeSearchDataHandler(s))
	mux.HandleFunc("/delete", makeDeleteDataHandler(s))
	mux.HandleFunc("/view", makeViewFileHandler(s))
}

func (s *Service) handleTextFile(ctx context.Context, f multipart.File, handler *multipart.FileHeader) error {
	objectName, err := s.getCurrentVersion(ctx, handler.Filename)
	if err != nil {
		return err
	}

	objectName, err = s.storage.UploadFile(ctx, f, objectName, handler.Header.Get("Content-Type"), handler.Size)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	err = s.repo.Insert(ctx, f, handler.Filename, objectName)
	if err != nil {
		return fmt.Errorf("insert form txt file: %w", err)
	}

	return nil
}

func (s *Service) handleZipFile(ctx context.Context, unzipper *zip.Reader) error {
	for _, file := range unzipper.File {
		if file.FileInfo().IsDir() || strings.Contains(file.Name, "MACOSX") || strings.Contains(file.Name, ".idea") {
			continue
		}

		fileName := getLastFile(file.Name)
		reader, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip file: %w", err)
		}
		var buf bytes.Buffer
		tee := io.TeeReader(reader, &buf)

		objectName, err := s.getCurrentVersion(ctx, fileName)
		if err != nil {
			return err
		}

		objectName, err = s.storage.UploadFile(ctx, tee, objectName, file.FileHeader.Mode().Type().String(), file.FileInfo().Size())
		if err != nil {
			return fmt.Errorf("upload zip file: %w", err)
		}

		err = s.repo.Insert(ctx, &buf, fileName, objectName)
		if err != nil {
			return fmt.Errorf("insert zip file: %w", err)
		}
	}

	return nil
}

func getLastFile(filePath string) string {
	spl := strings.Split(filePath, "/")
	if len(spl) == 0 {
		return "empty"
	}
	return spl[len(spl)-1]
}

func getFileSize(f io.Seeker) (int64, error) {
	fileSize, err := f.Seek(0, 2) //2 = from end
	if err != nil {
		return 0, err
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		return 0, err
	}

	return fileSize, nil
}

func (s *Service) getCurrentVersion(ctx context.Context, objectName string) (string, error) {
	err := s.repo.IncrementFileNameVersion(ctx, objectName)
	if err != nil {
		return "", fmt.Errorf("increment version: %w", err)
	}

	version, err := s.repo.GetFileNamesVersion(ctx, objectName)
	if err != nil {
		return "", fmt.Errorf("get file names version: %w", err)
	}

	objectName = fmt.Sprintf("%d.%s", version, objectName)

	return objectName, nil
}
