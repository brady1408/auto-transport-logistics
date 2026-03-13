package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	migcomp "github.com/brady1408/auto-transport-logistics/internal/handler/components/migration"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/riverargs"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type migrationCompanyStore interface {
	ListAll(ctx context.Context) ([]models.Company, error)
}

type MigrationHandler struct {
	runStore      *store.MigrationRunStore
	companyStore  migrationCompanyStore
	river         *river.Client[pgx.Tx]
	migrationsDir string
	deps          *Deps
}

func NewMigrationHandler(
	runStore *store.MigrationRunStore,
	companyStore migrationCompanyStore,
	riverClient *river.Client[pgx.Tx],
	migrationsDir string,
	deps *Deps,
) *MigrationHandler {
	return &MigrationHandler{
		runStore:      runStore,
		companyStore:  companyStore,
		river:         riverClient,
		migrationsDir: migrationsDir,
		deps:          deps,
	}
}

func (h *MigrationHandler) Register(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.HandleFunc("GET /admin/migration", h.index)
	mux.Handle("GET /admin/migration/new", wrap(h.newForm))
	mux.Handle("POST /admin/migration/start", wrap(h.start))
	mux.HandleFunc("GET /admin/migration/{id}", h.show)
	mux.HandleFunc("GET /admin/migration/{id}/log", h.logPoll)
	mux.Handle("POST /admin/migration/{id}/rerun", wrap(h.rerun))
}

func (h *MigrationHandler) index(w http.ResponseWriter, r *http.Request) {
	runs, err := h.runStore.List(r.Context())
	if err != nil {
		http.Error(w, "error loading runs", http.StatusInternalServerError)
		return
	}
	h.deps.renderTempl(w, r, migcomp.Index(h.deps.pageContext(w, r), runs))
}

func (h *MigrationHandler) newForm(w http.ResponseWriter, r *http.Request) {
	companies, err := h.companyStore.ListAll(r.Context())
	if err != nil {
		http.Error(w, "error loading companies", http.StatusInternalServerError)
		return
	}
	h.deps.renderTempl(w, r, migcomp.NewForm(h.deps.pageContext(w, r), companies))
}

func (h *MigrationHandler) start(w http.ResponseWriter, r *http.Request) {
	// 2GB max — .bak files can be large
	if err := r.ParseMultipartForm(2 << 30); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	companyID, err := strconv.Atoi(r.FormValue("company_id"))
	if err != nil || companyID == 0 {
		http.Error(w, "invalid company_id", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	run, err := h.runStore.Create(r.Context(), int64(companyID), header.Filename)
	if err != nil {
		http.Error(w, "error creating run", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(h.migrationsDir, 0755); err != nil {
		http.Error(w, "error creating migrations dir", http.StatusInternalServerError)
		return
	}
	bakPath := filepath.Join(h.migrationsDir, fmt.Sprintf("%d.bak", run.ID))
	f, err := os.Create(bakPath)
	if err != nil {
		http.Error(w, "error saving file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		http.Error(w, "error writing file", http.StatusInternalServerError)
		return
	}

	_, err = h.river.Insert(r.Context(), riverargs.MigrateArgs{
		RunID:     run.ID,
		CompanyID: companyID,
		BakPath:   bakPath,
	}, &river.InsertOpts{Queue: "migration"})
	if err != nil {
		http.Error(w, "error enqueueing job", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/migration/%d", run.ID), http.StatusSeeOther)
}

func (h *MigrationHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	run, err := h.runStore.Get(r.Context(), int64(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.deps.renderTempl(w, r, migcomp.Show(h.deps.pageContext(w, r), run))
}

func (h *MigrationHandler) logPoll(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	logText, status, err := h.runStore.GetLog(r.Context(), int64(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.deps.renderTempl(w, r, migcomp.LogPanel(logText, status))
}

func (h *MigrationHandler) rerun(w http.ResponseWriter, r *http.Request) {
	// TODO: implement data wipe + re-enqueue
	http.Redirect(w, r, "/admin/migration/new", http.StatusSeeOther)
}
