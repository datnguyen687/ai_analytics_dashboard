package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"analytics-dashboard-be/internal/domain"
	"analytics-dashboard-be/internal/service"
)

const ctxUserKey = "authUser"

// AuthHandler serves login and the current-user endpoint.
type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string                    `json:"token"`
	User  service.AuthenticatedUser `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		strings.TrimSpace(req.Username) == "" || req.Password == "" {
		fail(c, domain.NewAPIError(http.StatusBadRequest, "VALIDATION_ERROR", "username and password are required"))
		return
	}
	token, user, err := h.auth.Login(c.Request.Context(), strings.TrimSpace(strings.ToLower(req.Username)), req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, loginResponse{Token: token, User: user})
}

// Me returns the identity behind the current token (set by the auth middleware).
func (h *AuthHandler) Me(c *gin.Context) {
	u, ok := c.Get(ctxUserKey)
	if !ok {
		fail(c, domain.ErrTokenMissing)
		return
	}
	c.JSON(http.StatusOK, u)
}

// Users lists all accounts — admin-only (gated by the admin:manage claim on the
// route). Password hashes are never included.
func (h *AuthHandler) Users(c *gin.Context) {
	users, err := h.auth.Users(c.Request.Context())
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// RequireAuth validates the Bearer token and injects the identity into context.
func RequireAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			fail(c, domain.ErrTokenMissing)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		user, err := auth.Validate(token)
		if err != nil {
			fail(c, err)
			return
		}
		c.Set(ctxUserKey, user)
		c.Next()
	}
}

// RequireClaim gates a route on a specific claim. Must run after RequireAuth.
func RequireClaim(claim domain.Claim) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := c.Get(ctxUserKey)
		if !ok {
			fail(c, domain.ErrTokenMissing)
			return
		}
		user := u.(service.AuthenticatedUser)
		if !domain.HasClaim(user.Role, claim) {
			fail(c, domain.ErrForbidden)
			return
		}
		c.Next()
	}
}
