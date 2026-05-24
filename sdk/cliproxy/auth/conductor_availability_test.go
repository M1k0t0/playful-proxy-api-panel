package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestUpdateAggregatedAvailability_UnavailableWithoutNextRetryDoesNotBlockAuth(t *testing.T) {
	t.Parallel()

	now := time.Now()
	model := "test-model"
	auth := &Auth{
		ID: "a",
		ModelStates: map[string]*ModelState{
			model: {
				Status:      StatusError,
				Unavailable: true,
			},
		},
	}

	updateAggregatedAvailability(auth, now)

	if auth.Unavailable {
		t.Fatalf("auth.Unavailable = true, want false")
	}
	if !auth.NextRetryAfter.IsZero() {
		t.Fatalf("auth.NextRetryAfter = %v, want zero", auth.NextRetryAfter)
	}
}

func TestUpdateAggregatedAvailability_FutureNextRetryBlocksAuth(t *testing.T) {
	t.Parallel()

	now := time.Now()
	model := "test-model"
	next := now.Add(5 * time.Minute)
	auth := &Auth{
		ID: "a",
		ModelStates: map[string]*ModelState{
			model: {
				Status:         StatusError,
				Unavailable:    true,
				NextRetryAfter: next,
			},
		},
	}

	updateAggregatedAvailability(auth, now)

	if !auth.Unavailable {
		t.Fatalf("auth.Unavailable = false, want true")
	}
	if auth.NextRetryAfter.IsZero() {
		t.Fatalf("auth.NextRetryAfter = zero, want %v", next)
	}
	if auth.NextRetryAfter.Sub(next) > time.Second || next.Sub(auth.NextRetryAfter) > time.Second {
		t.Fatalf("auth.NextRetryAfter = %v, want %v", auth.NextRetryAfter, next)
	}
}

func TestManagerMarkResult_RequestScopedCodexTransientDoesNotBlockOnlyAuth(t *testing.T) {
	t.Parallel()

	model := "gpt-5.3-codex-spark"
	cases := []struct {
		name       string
		httpStatus int
		message    string
	}{
		{
			name:       "stream closed before completed",
			httpStatus: http.StatusRequestTimeout,
			message:    "stream error: stream disconnected before completion: stream closed before response.completed",
		},
		{
			name:       "upstream eof",
			httpStatus: http.StatusBadGateway,
			message:    `upstream transport error: Post "https://chatgpt.com/backend-api/codex/responses": EOF`,
		},
		{
			name:       "context canceled",
			httpStatus: http.StatusInternalServerError,
			message:    "context canceled",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			manager := NewManager(nil, nil, nil)
			if _, err := manager.Register(context.Background(), &Auth{
				ID:       "codex-auth",
				Provider: "codex",
				Status:   StatusActive,
			}); err != nil {
				t.Fatalf("register auth: %v", err)
			}

			manager.MarkResult(context.Background(), Result{
				AuthID:   "codex-auth",
				Provider: "codex",
				Model:    model,
				Success:  false,
				Error: &Error{
					HTTPStatus: tc.httpStatus,
					Message:    tc.message,
				},
			})

			updated, ok := manager.GetByID("codex-auth")
			if !ok || updated == nil {
				t.Fatalf("expected auth to be present")
			}
			state := updated.ModelStates[model]
			if state == nil {
				t.Fatalf("expected model state to be recorded")
			}
			if updated.Unavailable {
				t.Fatalf("auth.Unavailable = true, want false")
			}
			if state.Unavailable {
				t.Fatalf("state.Unavailable = true, want false")
			}
			if !state.NextRetryAfter.IsZero() {
				t.Fatalf("state.NextRetryAfter = %v, want zero", state.NextRetryAfter)
			}
		})
	}
}
