package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockDashboardHandler struct{}

func (m *mockDashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockDashboardHandler) HealthAPI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}

type mockAPIHandler struct{}

func (m *mockAPIHandler) Stats(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockAPIHandler) TargetsList(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockAPIHandler) Logs(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockAPIHandler) StartCrawling(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockAPIHandler) RefreshSitemaps(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}

type mockTargetsHandler struct{}

func (m *mockTargetsHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockTargetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}
func (m *mockTargetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	if _, err := w.Write([]byte("ok")); err != nil {
		panic(err)
	}
}

func TestServerRoutes(t *testing.T) {
	dh := &mockDashboardHandler{}
	ah := &mockAPIHandler{}
	th := &mockTargetsHandler{}
	s := NewWithHandlers(nil, dh, ah, th)
	handler := s.Handler()

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/", 200},
		{"GET", "/health", 200},
		{"GET", "/api/stats", 200},
		{"GET", "/api/targets", 200},
		{"GET", "/api/logs", 200},
		{"POST", "/api/crawl/start", 200},
		{"POST", "/api/sitemap/refresh-all", 200},
	}

	for _, tc := range tests {
		r := httptest.NewRequest(tc.method, tc.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, w.Code, tc.want)
		}
	}
}
