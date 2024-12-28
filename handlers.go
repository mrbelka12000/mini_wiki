package mini_wiki

import (
	"archive/zip"
	"database/sql"
	"errors"
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

				err = s.handleZipFile(r.Context(), unzipper, handler.Filename)
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

			case strings.Contains(fileType, "application/pdf"):
				err := s.handlePDFFile(r.Context(), f, handler)
				if err != nil {
					s.log.With("error", err).Error("error handling pdf")
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
		if objectName != "" {
			err = s.repo.Delete(r.Context(), objectName)
			if err != nil {
				s.log.With("error", err).Error("error deleting object")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		objectNamePDF := r.FormValue("object_name_pdf")
		if objectNamePDF != "" {
			err = s.repo.DeletePDF(r.Context(), objectNamePDF)
			if err != nil {
				s.log.With("error", err).Error("error deleting object")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if objectName == "" {
			objectName = objectNamePDF
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

func makeSearchCodeHandler(s *Service) http.HandlerFunc {
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
				Files  []filesResponse
				ToFind string
			}{
				Files:  files,
				ToFind: toFind,
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

func makeSearchPDFHandler(s *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("html_templates/search_pdf.html")
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

			var errorMessage string
			toFind := r.FormValue("to_find")
			pdfName, err := s.repo.FindPDF(r.Context(), toFind)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					s.log.With("error", err).Error("error finding pdfName")
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				errorMessage = "PDF not found"
			}

			data := struct {
				Files        []string
				ToFind       string
				ErrorMessage string
			}{
				Files:        pdfName,
				ToFind:       toFind,
				ErrorMessage: errorMessage,
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
		if objectName != "" {
			err = s.storage.DownloadFile(r.Context(), w, objectName, "")
			if err != nil {
				s.log.With("error", err).Error("error downloading file")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			return
		}

		pdfName := r.Form.Get("pdf_name")
		if pdfName != "" {
			err = s.storage.DownloadFile(r.Context(), w, pdfName, "application/pdf")
			if err != nil {
				s.log.With("error", err).Error("error downloading pdf file")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			return
		}

		http.Error(w, "no file found", http.StatusBadRequest)
		s.log.Error("no file found")
	}
}
