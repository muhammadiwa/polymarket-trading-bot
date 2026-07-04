package circuitbreaker_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	circuitbreaker "github.com/pqap/services/execution-engine/internal/circuit_breaker"
	"go.uber.org/zap"
)

func TestResumeHandler_Success(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	handler := circuitbreaker.NewResumeHandler(cb, "test-secret", log)

	body := `{"reason":"manual intervention"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/circuit-breaker/resume", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp circuitbreaker.ResumeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "resumed" {
		t.Errorf("status = %s, want resumed", resp.Status)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("state = %d, want closed", cb.GetState())
	}
}

func TestResumeHandler_MethodNotAllowed(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	handler := circuitbreaker.NewResumeHandler(cb, "test-secret", log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution/circuit-breaker/resume", nil)
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestResumeHandler_AuthBypassRejected(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	handler := circuitbreaker.NewResumeHandler(cb, "", log)

	body := `{"reason":"manual intervention"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/circuit-breaker/resume", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (auth bypass should be rejected when secret is empty)", w.Code, http.StatusUnauthorized)
	}
}

func TestResumeHandler_NotOpen(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	handler := circuitbreaker.NewResumeHandler(cb, "test-secret", log)

	body := `{"reason":"manual intervention"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/circuit-breaker/resume", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestResumeHandler_WithJWT(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	handler := circuitbreaker.NewResumeHandler(cb, "test-secret", log)

	body := `{"reason":"manual intervention"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/circuit-breaker/resume", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestResumeHandler_Unauthorized(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	handler := circuitbreaker.NewResumeHandler(cb, "test-secret", log)

	body := `{"reason":"manual intervention"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/execution/circuit-breaker/resume", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleResume(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestStatusHandler_Success(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	handler := circuitbreaker.NewResumeHandler(cb, "", log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution/circuit-breaker/status", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp circuitbreaker.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.State != "CLOSED" {
		t.Errorf("state = %s, want CLOSED", resp.State)
	}

	if resp.ConsecutiveErrors != 0 {
		t.Errorf("consecutive errors = %d, want 0", resp.ConsecutiveErrors)
	}
}

func TestStatusHandler_Open(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mockPub := &mockPublisherBreaker{}

	cb, err := circuitbreaker.NewCircuitBreaker(2, 1*time.Second, 5*time.Second, log, mockPub, nil)
	if err != nil {
		t.Fatalf("unexpected error creating circuit breaker: %v", err)
	}

	testErr := errors.New("test error")
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	handler := circuitbreaker.NewResumeHandler(cb, "", log)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/execution/circuit-breaker/status", nil)
	w := httptest.NewRecorder()

	handler.HandleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp circuitbreaker.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.State != "OPEN" {
		t.Errorf("state = %s, want OPEN", resp.State)
	}

	if resp.ConsecutiveErrors != 2 {
		t.Errorf("consecutive errors = %d, want 2", resp.ConsecutiveErrors)
	}

	if resp.LastError != "test error" {
		t.Errorf("last error = %s, want test error", resp.LastError)
	}
}
