package circuitbreaker

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type ResumeRequest struct {
	Reason string `json:"reason"`
}

type ResumeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type StatusResponse struct {
	State               string  `json:"state"`
	ConsecutiveErrors   int     `json:"consecutive_errors"`
	LastError           string  `json:"last_error"`
	LastErrorTime       string  `json:"last_error_time"`
	TrippedAt           string  `json:"tripped_at,omitempty"`
	ResumedAt           string  `json:"resumed_at,omitempty"`
	TotalTrips          int64   `json:"total_trips"`
	CooldownRemainingMs int64   `json:"cooldown_remaining_ms"`
}

type ResumeHandler struct {
	breaker   *CircuitBreaker
	jwtSecret string
	logger    *zap.Logger
}

func NewResumeHandler(breaker *CircuitBreaker, jwtSecret string, logger *zap.Logger) *ResumeHandler {
	return &ResumeHandler{
		breaker:   breaker,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

func (rh *ResumeHandler) HandleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		rh.writeJSONError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := rh.authenticate(r)
	if err != nil {
		rh.logger.Warn("authentication failed", zap.Error(err))
		rh.writeJSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
	}

	var req ResumeRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rh.writeJSONError(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if err := rh.breaker.Resume(req.Reason, userID); err != nil {
		rh.logger.Error("failed to resume circuit breaker",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		rh.writeJSONError(w, "circuit breaker is not in OPEN state", http.StatusConflict)
		return
	}

	rh.logger.Info("circuit breaker resumed via API",
		zap.String("user_id", userID),
		zap.String("reason", req.Reason),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ResumeResponse{
		Status:  "resumed",
		Message: "Circuit breaker resumed successfully. Trading will resume on next opportunity.",
	})
}

func (rh *ResumeHandler) writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (rh *ResumeHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	state := rh.breaker.GetState()
	stateStr := "CLOSED"
	switch state {
	case StateOpen:
		stateStr = "OPEN"
	case StateHalfOpen:
		stateStr = "HALF_OPEN"
	}

	lastErrorTime := ""
	if !rh.breaker.GetLastErrorTime().IsZero() {
		lastErrorTime = rh.breaker.GetLastErrorTime().Format(time.RFC3339)
	}

	trippedAt := ""
	if t := rh.breaker.GetTrippedAt(); t != nil {
		trippedAt = t.Format(time.RFC3339)
	}

	resumedAt := ""
	if t := rh.breaker.GetResumedAt(); t != nil {
		resumedAt = t.Format(time.RFC3339)
	}

	resp := StatusResponse{
		State:               stateStr,
		ConsecutiveErrors:   rh.breaker.GetConsecutiveErrors(),
		LastError:           rh.breaker.GetLastError(),
		LastErrorTime:       lastErrorTime,
		TrippedAt:           trippedAt,
		ResumedAt:           resumedAt,
		TotalTrips:          rh.breaker.GetTotalTrips(),
		CooldownRemainingMs: rh.breaker.GetCooldownRemaining().Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (rh *ResumeHandler) authenticate(r *http.Request) (string, error) {
	if rh.jwtSecret == "" {
		return "", fmt.Errorf("JWT secret not configured; authentication required")
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("empty token")
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(rh.jwtSecret)) != 1 {
		return "", fmt.Errorf("invalid token")
	}

	return "authenticated-user", nil
}
