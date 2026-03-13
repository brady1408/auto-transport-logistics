package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/audit"
	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

// testDeps returns a Deps suitable for unit tests (no real DB needed).
func testDeps() *Deps {
	return &Deps{Audit: audit.NewService(nil)}
}

// mockFeedbackStore satisfies the feedbackStore interface for tests.
type mockFeedbackStore struct {
	listResult *models.FeedbackListResult
	listErr    error
	createErr  error
	created    *models.Feedback
}

func (m *mockFeedbackStore) List(_ context.Context, _ models.FeedbackFilter) (*models.FeedbackListResult, error) {
	if m.listResult != nil {
		return m.listResult, m.listErr
	}
	return &models.FeedbackListResult{Items: []models.Feedback{}, TotalCount: 0, Page: 1, PageSize: 25}, m.listErr
}

func (m *mockFeedbackStore) GetByID(_ context.Context, _ int) (*models.Feedback, error) {
	return nil, nil
}

func (m *mockFeedbackStore) Create(_ context.Context, fb *models.Feedback) error {
	if m.createErr != nil {
		return m.createErr
	}
	fb.ID = 42
	fb.Status = "open"
	fb.CreatedAt = time.Now()
	fb.UpdatedAt = time.Now()
	m.created = fb
	return nil
}

func (m *mockFeedbackStore) Update(_ context.Context, _ *models.Feedback) error { return nil }
func (m *mockFeedbackStore) Delete(_ context.Context, _ int) error               { return nil }
func (m *mockFeedbackStore) ListComments(_ context.Context, _ int, _ bool) ([]models.FeedbackComment, error) {
	return nil, nil
}
func (m *mockFeedbackStore) CreateComment(_ context.Context, _ *models.FeedbackComment) error {
	return nil
}

// ctxWithUser injects a ContextUser so handler methods can call auth.GetUser / auth.GetCompanyID.
func ctxWithUser(r *http.Request, userID, companyID int, username, role string) *http.Request {
	ctx := auth.SetUser(r.Context(), auth.ContextUser{
		ID:        userID,
		Username:  username,
		Role:      role,
		CompanyID: companyID,
	})
	return r.WithContext(ctx)
}

func TestFeedbackAPI_List(t *testing.T) {
	store := &mockFeedbackStore{
		listResult: &models.FeedbackListResult{
			Items:      []models.Feedback{{ID: 1, Message: "hello"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   25,
		},
	}
	h := NewFeedbackAPIHandler(store, testDeps())

	mux := http.NewServeMux()
	h.Register(mux)

	r := httptest.NewRequest("GET", "/api/feedback", nil)
	r = ctxWithUser(r, 1, 2, "api", "super_admin")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if total := body["total"].(float64); total != 1 {
		t.Errorf("total = %v, want 1", total)
	}
}

func TestFeedbackAPI_Create_Success(t *testing.T) {
	store := &mockFeedbackStore{}
	h := NewFeedbackAPIHandler(store, testDeps())

	mux := http.NewServeMux()
	h.Register(mux)

	payload := map[string]string{"category": "feature", "message": "please add dark mode"}
	body, _ := json.Marshal(payload)
	r := httptest.NewRequest("POST", "/api/feedback", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctxWithUser(r, 1, 2, "api", "super_admin")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"].(float64) != 42 {
		t.Errorf("id = %v, want 42", resp["id"])
	}
	if store.created == nil {
		t.Fatal("expected store.Create to be called")
	}
	if store.created.Message != "please add dark mode" {
		t.Errorf("message = %q", store.created.Message)
	}
}

func TestFeedbackAPI_Create_MissingMessage(t *testing.T) {
	store := &mockFeedbackStore{}
	h := NewFeedbackAPIHandler(store, testDeps())

	mux := http.NewServeMux()
	h.Register(mux)

	payload := map[string]string{"category": "bug"}
	body, _ := json.Marshal(payload)
	r := httptest.NewRequest("POST", "/api/feedback", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctxWithUser(r, 1, 2, "api", "super_admin")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestFeedbackAPI_Create_DefaultCategory(t *testing.T) {
	store := &mockFeedbackStore{}
	h := NewFeedbackAPIHandler(store, testDeps())

	mux := http.NewServeMux()
	h.Register(mux)

	payload := map[string]string{"message": "just a note"}
	body, _ := json.Marshal(payload)
	r := httptest.NewRequest("POST", "/api/feedback", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctxWithUser(r, 1, 2, "api", "super_admin")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if store.created.Category != "other" {
		t.Errorf("default category = %q, want 'other'", store.created.Category)
	}
}
