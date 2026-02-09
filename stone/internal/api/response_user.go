package api

import (
	"time"

	"github.com/pulse/stone/internal/model"
)

type selfUserResponse struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Location    string `json:"location"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toSelfUserResponse(user *model.User) selfUserResponse {
	return selfUserResponse{
		ID:          user.ID.String(),
		Handle:      user.Handle,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Bio:         user.Bio,
		AvatarURL:   user.AvatarURL,
		Location:    user.Location,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339Nano),
	}
}
