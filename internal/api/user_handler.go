package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/identity"
)

// UserHandler handles user-related HTTP endpoints.
type UserHandler struct {
	authService *identity.AuthService
	pool        *identity.DBPool
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(authService *identity.AuthService, pool *identity.DBPool) *UserHandler {
	return &UserHandler{authService: authService, pool: pool}
}

// UserProfileResponse represents the /api/v1/user/profile response.
type UserProfileResponse struct {
	User     UserProfileData       `json:"user"`
	Usage    UsageSummaryResponse  `json:"usage"`
	Quota    QuotaResponse         `json:"quota"`
	Metering bool                  `json:"metering_active"`
}

// UserProfileData contains user profile information.
type UserProfileData struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	RoleName  string    `json:"role_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// UsageSummaryResponse contains usage statistics.
type UsageSummaryResponse struct {
	BooksUploaded int     `json:"books_uploaded"`
	TTSMinutes    float64 `json:"tts_minutes"`
	StorageBytes  int64   `json:"storage_bytes"`
	TotalSegments int     `json:"total_segments"`
}

// QuotaResponse contains quota limits and consumption.
type QuotaResponse struct {
	Plan       string  `json:"plan"`
	TTSMinutes float64 `json:"tts_minutes_limit"`
	TTSUsed    float64 `json:"tts_minutes_used"`
	StorageGB  float64 `json:"storage_gb_limit"`
	StorageUsed float64 `json:"storage_gb_used"`
	BooksLimit int     `json:"books_limit"`
	BooksUsed  int     `json:"books_used"`
}

// Profile handles GET /api/v1/user/profile - returns current user profile with usage and quota.
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowedError(w, r)
		return
	}

	user := GetUserFromContext(r.Context())
	if user == nil {
		WriteStructuredError(w, r, ErrCodeUnauthorized, "not authenticated", http.StatusUnauthorized)
		return
	}

	// Resolve role name
	roleName := ""
	if user.RoleID != "" && h.pool != nil {
		role, err := h.pool.Roles.GetRoleByID(r.Context(), user.RoleID)
		if err == nil && role != nil {
			roleName = role.Name
		}
	}

	// TODO(Milestone 3): Replace with real usage metering and quota enforcement.
	// For now, return placeholder data since usage metering is not yet implemented.
	resp := UserProfileResponse{
		User: UserProfileData{
			ID:        user.ID,
			AccountID: user.AccountID,
			Email:     user.Email,
			Name:      user.Name,
			RoleName:  roleName,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
		},
		Usage: UsageSummaryResponse{
			BooksUploaded: 0,
			TTSMinutes:    0,
			StorageBytes:  0,
			TotalSegments: 0,
		},
		Quota: QuotaResponse{
			Plan:        "free",
			TTSMinutes:  -1, // -1 means unlimited/no quota set yet
			TTSUsed:     0,
			StorageGB:   -1,
			StorageUsed: 0,
			BooksLimit:  -1,
			BooksUsed:   0,
		},
		Metering: false, // Usage metering not active until Milestone 3
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
