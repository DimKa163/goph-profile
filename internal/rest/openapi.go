package rest

import (
	"github.com/DimKa163/goph-profile/internal/openapi"
	"github.com/labstack/echo/v5"
)

// OpenAPIHandler open api handler
type OpenAPIHandler struct {
	avatar *avatarController
	user   *userController
}

var _ openapi.ServerInterface = (*OpenAPIHandler)(nil)

// NewOpenAPIHandler ctor
func NewOpenAPIHandler(
	avatar *avatarController,
	user *userController,
) *OpenAPIHandler {
	return &OpenAPIHandler{
		avatar: avatar,
		user:   user,
	}
}

// UploadAvatar upload avatar
func (h *OpenAPIHandler) UploadAvatar(c *echo.Context, _ openapi.UploadAvatarParams) error {
	return h.avatar.Avatar(c)
}

// DeleteAvatar delete avatar
func (h *OpenAPIHandler) DeleteAvatar(ctx *echo.Context, _ string, _ openapi.DeleteAvatarParams) error {
	return h.avatar.Delete(ctx)
}

// GetAvatar get avatar
func (h *OpenAPIHandler) GetAvatar(ctx *echo.Context, _ string, _ openapi.GetAvatarParams) error {
	return h.avatar.Get(ctx)
}

// GetAvatarMetadata metadata
func (h *OpenAPIHandler) GetAvatarMetadata(ctx *echo.Context, _ string, _ openapi.GetAvatarMetadataParams) error {
	return h.avatar.Metadata(ctx)
}

// DeleteUserAvatar delete avatar
func (h *OpenAPIHandler) DeleteUserAvatar(ctx *echo.Context, _ string) error {
	return h.user.Delete(ctx)
}

// GetCurrentUserAvatar current user avatar
func (h *OpenAPIHandler) GetCurrentUserAvatar(ctx *echo.Context, _ string, _ openapi.GetCurrentUserAvatarParams) error {
	return h.user.Avatar(ctx)
}

func (h *OpenAPIHandler) GetAllUserAvatars(ctx *echo.Context, userId string) error {
	return h.user.Avatars(ctx)
}
