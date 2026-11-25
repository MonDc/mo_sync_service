package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mondc/mo_sync_service/internal/application/commands"
	"github.com/mondc/mo_sync_service/internal/application/queries"
	"github.com/mondc/mo_sync_service/internal/domain/errors"
	"github.com/mondc/mo_sync_service/internal/infrastructure/observability/metrics"
	"github.com/mondc/mo_sync_service/internal/infrastructure/observability/tracing"
	"go.uber.org/zap"
)

type UserHandler struct {
	syncUserHandler *commands.SyncUserHandler
	getUserHandler  *queries.GetUserHandler
	logger          *zap.Logger
	metrics         *metrics.Metrics
}

func NewUserHandler(
	syncUserHandler *commands.SyncUserHandler,
	getUserHandler *queries.GetUserHandler,
	logger *zap.Logger,
	metrics *metrics.Metrics,
) *UserHandler {
	return &UserHandler{
		syncUserHandler: syncUserHandler,
		getUserHandler:  getUserHandler,
		logger:          logger,
		metrics:         metrics,
	}
}

// SyncUser godoc
// @Summary Sync a user by ID
// @Description Sync user from main database to local database
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} user.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{id}/sync [post]
func (h *UserHandler) SyncUser(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.StartSpan(r.Context(), "SyncUser")
	defer span.End()

	vars := mux.Vars(r)
	userID := vars["id"]

	h.metrics.ActiveSyncs.Inc()
	defer h.metrics.ActiveSyncs.Dec()

	start := time.Now()
	defer h.metrics.UserSyncDuration.WithLabelValues("sync_user").Observe(time.Since(start).Seconds())

	cmd := commands.SyncUserCommand{UserID: userID}
	user, err := h.syncUserHandler.Handle(ctx, cmd)
	if err != nil {
		h.metrics.UserSyncErrors.WithLabelValues("sync_failed").Inc()
		h.handleError(w, err)
		return
	}

	h.metrics.UserSyncTotal.WithLabelValues("success").Inc()
	h.logger.Info("user synced successfully",
		zap.String("user_id", userID),
		zap.Time("synced_at", *user.SyncedAt),
	)

	h.respondWithJSON(w, http.StatusOK, user)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user from local database
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} user.User
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.StartSpan(r.Context(), "GetUser")
	defer span.End()

	vars := mux.Vars(r)
	userID := vars["id"]

	query := queries.GetUserQuery{UserID: userID}
	user, err := h.getUserHandler.Handle(ctx, query)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondWithJSON(w, http.StatusOK, user)
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Get service health status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *UserHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "mo_sync_service",
	})
}

func (h *UserHandler) handleError(w http.ResponseWriter, err error) {
	h.logger.Error("request failed", zap.Error(err))

	var statusCode int
	var response map[string]string

	switch e := err.(type) {
	case *errors.DomainError:
		switch e.Code {
		case "USER_NOT_FOUND":
			statusCode = http.StatusNotFound
		case "USER_ALREADY_EXISTS":
			statusCode = http.StatusConflict
		case "INVALID_USER_DATA":
			statusCode = http.StatusBadRequest
		default:
			statusCode = http.StatusInternalServerError
		}
		response = map[string]string{
			"error":   e.Code,
			"message": e.Message,
		}
	default:
		statusCode = http.StatusInternalServerError
		response = map[string]string{
			"error":   "INTERNAL_SERVER_ERROR",
			"message": "an unexpected error occurred",
		}
	}

	h.respondWithJSON(w, statusCode, response)
}

func (h *UserHandler) respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}