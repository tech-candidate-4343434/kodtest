package api

import (
	"bytes"
	"database/sql"
	"embed"
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tech-candidate-4343434/guestbook/internal/store"
)

var (
	//go:embed templates
	templateFS embed.FS

	contentTmpl  = parsePage("content.tmpl")
	notFoundTmpl = parsePage("notfound.tmpl")
)

var funcs = template.FuncMap{
	"fmtTime": func(t sql.NullTime) string {
		if !t.Valid {
			return ""
		}
		return t.Time.Format("2006-01-02 15:04")
	},
}

func parsePage(page string) *template.Template {
	return template.Must(template.New("base.tmpl").Funcs(funcs).
		ParseFS(templateFS, "templates/base.tmpl", "templates/"+page))
}

func render(w http.ResponseWriter, r *http.Request, tmpl *template.Template, status int, data any) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		slog.ErrorContext(r.Context(), "render page", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

type Handler struct {
	queries *store.Queries
}

func NewHandler(queries *store.Queries) *Handler {
	return &Handler{queries: queries}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.Index)
	mux.HandleFunc("POST /entries", h.CreateEntry)
	mux.HandleFunc("/", h.NotFound)
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	entries, err := h.queries.ListEntries(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "list entries", "error", err)
		http.Error(w, "failed to load guestbook", http.StatusInternalServerError)
		return
	}

	render(w, r, contentTmpl, http.StatusOK, entries)
}

func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form submission", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.PostFormValue("name"))
	message := strings.TrimSpace(r.PostFormValue("message"))

	if name == "" || message == "" {
		http.Error(w, "name and message are required", http.StatusBadRequest)
		return
	}

	if err := h.queries.CreateEntry(ctx, store.CreateEntryParams{
		Name:    name,
		Message: message,
	}); err != nil {
		slog.ErrorContext(ctx, "create entry", "error", err)
		http.Error(w, "failed to create entry", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	render(w, r, notFoundTmpl, http.StatusNotFound, nil)
}
