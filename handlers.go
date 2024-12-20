package mini_wiki

import (
	"archive/zip"
	"html/template"
	"io"
	"net/http"
	"strings"
)

func makeIndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("html_templates/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func makeUploadDataHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(20000000)
		if err != nil {
			s.log.With("error", err).Error("error parsing form")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Get the file from the form
		file, handler, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			// Read the first 512 bytes to detect content type
			buffer := make([]byte, 512)
			_, err = file.Read(buffer)
			if err != nil && err != io.EOF {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}
			fileType := http.DetectContentType(buffer)

			f, err := handler.Open()
			if err != nil {
				s.log.With("error", err).Error("error opening file")
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}

			switch {
			case fileType == "application/zip":

				fileSize, err := getFileSize(f)
				unzipper, err := zip.NewReader(f, fileSize)
				if err != nil {
					s.log.With("error", err).Error("error unzipping file")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				err = s.handleZipFile(r.Context(), unzipper)
				if err != nil {
					s.log.With("error", err).Error("error handling zip file")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case strings.Contains(fileType, "text/plain"),
				strings.Contains(fileType, "application/octet-stream"):
				err := s.handleTextFile(r.Context(), f, handler)
				if err != nil {
					s.log.With("error", err).Error("error handling text")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case fileType == "text/html":
			}

			return
		}
	}
}

func makeDeleteDataHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseForm()
		if err != nil {
			s.log.With("error", err).Error("error parsing form")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		objectName := r.FormValue("object_name")

		err = s.repo.Delete(r.Context(), objectName)
		if err != nil {
			s.log.With("error", err).Error("error deleting object")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		t, err := template.ParseFiles("html_templates/delete.html")
		if err != nil {
			s.log.With("error", err).Error("error parsing template")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, objectName)
		if err != nil {
			s.log.With("error", err).Error("error executing template")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func makeSearchDataHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("html_templates/search.html")
		if err != nil {
			s.log.With("error", err).Error("error parsing template")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch r.Method {
		case http.MethodGet:

			err = t.Execute(w, nil)
			if err != nil {
				s.log.With("error", err).Error("error executing template")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodPost:

			err := r.ParseForm()
			if err != nil {
				s.log.With("error", err).Error("error parsing form")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			toFind := r.FormValue("to_find")
			files, err := s.repo.Find(r.Context(), toFind)
			if err != nil {
				s.log.With("error", err).Error("error finding files")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			data := struct {
				Files []string
			}{
				Files: files,
			}

			err = t.Execute(w, data)
			if err != nil {
				s.log.With("error", err).Error("error executing template")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func makeViewFileHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			s.log.With("error", err).Error("error parsing form")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		objectName := r.Form.Get("object_name")
		if objectName == "" {
			s.log.Error("empty object_name")
			http.Error(w, "empty object_name", http.StatusBadRequest)
			return
		}

		err = s.storage.DownloadFile(r.Context(), w, objectName)
		if err != nil {
			s.log.With("error", err).Error("error downloading file")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
