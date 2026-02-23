package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
)

type Deps struct {
	JWT      *auth.JWTService
	Audit    *audit.Service
	Tmpl     *TemplateMap
}

func parseID(r *http.Request) (int, error) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", idStr)
	}
	return id, nil
}

func parsePathID(r *http.Request, param string) (int, error) {
	idStr := r.PathValue(param)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", param, idStr)
	}
	return id, nil
}

func formDate(r *http.Request, key string) *time.Time {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil
	}
	return &t
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (d *Deps) render(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}

	// Add user to all templates
	if user, ok := auth.GetUserFromRequest(r); ok {
		data["User"] = user
	}

	// Add flash message if present
	if flash := getFlash(w, r); flash != "" {
		data["Flash"] = flash
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Check if this is a page template (renders with base layout)
	// or a standalone template (like login.html)
	var err error
	if _, ok := d.Tmpl.pages[page]; ok {
		err = d.Tmpl.RenderPage(w, page, data)
	} else {
		err = d.Tmpl.RenderTemplate(w, page, data)
	}
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (d *Deps) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := d.Tmpl.RenderTemplate(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    msg,
		Path:     "/",
		MaxAge:   5,
		HttpOnly: true,
	})
}

func getFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	// Clear the flash cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	return cookie.Value
}

// Form helpers
func formString(r *http.Request, key string) *string {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	return &v
}

func formStringRequired(r *http.Request, key string) string {
	return strings.TrimSpace(r.FormValue(key))
}

func formInt(r *http.Request, key string) *int {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &i
}

func formBool(r *http.Request, key string) bool {
	v := r.FormValue(key)
	return v == "on" || v == "true" || v == "1"
}

// writeCSV writes a CSV file to the response.
func writeCSV(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	writer := csv.NewWriter(w)
	writer.Write(headers)
	for _, row := range rows {
		writer.Write(row)
	}
	writer.Flush()
}
