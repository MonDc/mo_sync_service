package commands

import (
	"context"

	"github.com/mondc/mo_sync_service/internal/application/ports"
	"github.com/mondc/mo_sync_service/internal/domain/user"
)

type SyncUserCommand struct {
	UserID string
}

type SyncUserHandler struct {
	userService ports.UserSyncService
}

func NewSyncUserHandler(userService ports.UserSyncService) *SyncUserHandler {
	return &SyncUserHandler{
		userService: userService,
	}
}

func (h *SyncUserHandler) Handle(ctx context.Context, cmd SyncUserCommand) (*user.User, error) {
	return h.userService.SyncUser(ctx, cmd.UserID)
}
