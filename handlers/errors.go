package handlers

import (
	"html/template"
	"net/http"
)

func RenderError(w http.ResponseWriter, statusCode int) {
	var templateFile string

	switch statusCode {
	case http.StatusUnauthorized: // 401
		templateFile = "templates/401.html"
	case http.StatusForbidden: // 403
		templateFile = "templates/403.html"
	case http.StatusNotFound: // 404
		templateFile = "templates/404.html"
	case http.StatusInternalServerError: // 500
		templateFile = "templates/500.html"
	default:
		// Fallback to plain text error for other status codes
		http.Error(w, http.StatusText(statusCode), statusCode)
		return
	}

	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		// If template fails to load, fallback to plain text
		http.Error(w, http.StatusText(statusCode), statusCode)
		return
	}

	w.WriteHeader(statusCode)
	err = tmpl.Execute(w, nil)
	if err != nil {
		// If template execution fails, log it but don't try to send another response
		// as headers have already been written
		return
	}
}
