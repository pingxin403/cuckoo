package storage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPGetOfflineMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := &OfflineStore{db: db}
	handler := NewHTTPHandler(store)

	t.Run("successful retrieval", func(t *testing.T) {
		userID := "user-001"

		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		}).
			AddRow(1, "msg-001", userID, "user-002", "conv-001",
				"private", "Hello", 1, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour)).
			AddRow(2, "msg-002", userID, "user-003", "conv-002",
				"group", "World", 2, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour))

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs(userID, int64(0), 100).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?cursor=0&limit=100", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response OfflineMessagesResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Len(t, response.Messages, 2)
		assert.Equal(t, "msg-001", response.Messages[0].MsgID)
		assert.Equal(t, "msg-002", response.Messages[1].MsgID)
	})

	t.Run("missing user_id in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "unauthorized", response.Error)
	})

	t.Run("invalid cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?cursor=invalid", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-001"))
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "invalid_cursor", response.Error)
	})

	t.Run("invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=200", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-001"))
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "invalid_limit", response.Error)
	})

	t.Run("storage error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs("user-001", int64(0), 100).
			WillReturnError(assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-001"))
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response ErrorResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "storage_error", response.Error)
	})

	t.Run("pagination with has_more", func(t *testing.T) {
		userID := "user-001"
		limit := 2

		rows := sqlmock.NewRows([]string{
			"id", "msg_id", "user_id", "sender_id", "conversation_id",
			"conversation_type", "content", "sequence_number", "timestamp",
			"created_at", "expires_at",
		}).
			AddRow(1, "msg-001", userID, "user-002", "conv-001",
				"private", "Hello", 1, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour)).
			AddRow(2, "msg-002", userID, "user-003", "conv-002",
				"group", "World", 2, time.Now().Unix()*1000,
				time.Now(), time.Now().Add(7*24*time.Hour))

		mock.ExpectQuery("SELECT (.+) FROM offline_messages").
			WithArgs(userID, int64(0), limit).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=2", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
		w := httptest.NewRecorder()

		handler.GetOfflineMessages(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response OfflineMessagesResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.True(t, response.HasMore)
		assert.Equal(t, int64(2), response.NextCursor)
	})
}

func TestHTTPGetMessageCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := &OfflineStore{db: db}
	handler := NewHTTPHandler(store)

	t.Run("successful count", func(t *testing.T) {
		userID := "user-001"
		expectedCount := int64(42)

		rows := sqlmock.NewRows([]string{"count"}).AddRow(expectedCount)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages").
			WithArgs(userID).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline/count", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
		w := httptest.NewRecorder()

		handler.GetMessageCount(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response MessageCountResponse
		err := json.NewDecoder(w.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, expectedCount, response.Count)
	})

	t.Run("missing user_id in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline/count", nil)
		w := httptest.NewRecorder()

		handler.GetMessageCount(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("storage error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM offline_messages").
			WithArgs("user-001").
			WillReturnError(assert.AnError)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline/count", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user_id", "user-001"))
		w := httptest.NewRecorder()

		handler.GetMessageCount(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	auth := NewAuthMiddleware()

	t.Run("successful authentication", func(t *testing.T) {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			userID := r.Context().Value("user_id").(string)
			assert.Equal(t, "user-001", userID)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		req.Header.Set("Authorization", "Bearer user-001")
		w := httptest.NewRecorder()

		handler := auth.Authenticate(next)
		handler.ServeHTTP(w, req)

		assert.True(t, nextCalled)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		w := httptest.NewRecorder()

		handler := auth.Authenticate(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		handler := auth.Authenticate(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("empty token", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		handler := auth.Authenticate(next)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestParseCursor(t *testing.T) {
	handler := &HTTPHandler{}

	t.Run("valid cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?cursor=123", nil)
		cursor, err := handler.parseCursor(req)
		assert.NoError(t, err)
		assert.Equal(t, int64(123), cursor)
	})

	t.Run("missing cursor defaults to 0", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		cursor, err := handler.parseCursor(req)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), cursor)
	})

	t.Run("invalid cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?cursor=invalid", nil)
		_, err := handler.parseCursor(req)
		assert.Error(t, err)
	})

	t.Run("negative cursor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?cursor=-1", nil)
		_, err := handler.parseCursor(req)
		assert.Error(t, err)
	})
}

func TestParseLimit(t *testing.T) {
	handler := &HTTPHandler{}

	t.Run("valid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=50", nil)
		limit, err := handler.parseLimit(req)
		assert.NoError(t, err)
		assert.Equal(t, 50, limit)
	})

	t.Run("missing limit defaults to 100", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline", nil)
		limit, err := handler.parseLimit(req)
		assert.NoError(t, err)
		assert.Equal(t, 100, limit)
	})

	t.Run("invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=invalid", nil)
		_, err := handler.parseLimit(req)
		assert.Error(t, err)
	})

	t.Run("limit too small", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=0", nil)
		_, err := handler.parseLimit(req)
		assert.Error(t, err)
	})

	t.Run("limit too large", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/offline?limit=101", nil)
		_, err := handler.parseLimit(req)
		assert.Error(t, err)
	})
}
