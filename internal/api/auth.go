package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/arthurschuelter/habits-tracker/internal/auth"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type loginRequestDTO struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required" format:"password"`
}

type loginResponseDTO struct {
	Token string `json:"token"`
}

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Routes(e *echo.Echo) {
	limit := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 1, Burst: 5}),
		IdentifierExtractor: func(c *echo.Context) (string, error) {
			return echo.ExtractIPDirect()(c.Request()), nil
		},
	})
	e.POST("/api/v2/login", h.Login, limit, middleware.BodyLimit(8*1024))
}

// Login exchanges credentials for a session token.
// @Summary Sign in
// @Description Sign in with an existing username and password. The bearer token expires after 24 hours. Signing in again replaces the previous session for that user.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body loginRequestDTO true "Login credentials"
// @Success 200 {object} loginResponseDTO
// @Failure 400 {object} errorResponseDTO "Invalid request body"
// @Failure 401 {object} errorResponseDTO "Invalid username or password"
// @Failure 413 {object} errorResponseDTO "Request body too large"
// @Failure 429 {object} errorResponseDTO "Too many attempts"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /login [post]
func (h *AuthHandler) Login(c *echo.Context) error {
	var req loginRequestDTO
	if err := c.Bind(&req); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) && httpErr.Code == http.StatusRequestEntityTooLarge {
			return err
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	token, err := h.service.Login(c.Request().Context(), req.Username, req.Password)
	switch {
	case errors.Is(err, auth.ErrCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, auth.ErrCredentials.Error())
	case errors.Is(err, auth.ErrBusy):
		return echo.NewHTTPError(http.StatusTooManyRequests, auth.ErrBusy.Error())
	case err != nil:
		return fmt.Errorf("sign in: %w", err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(http.StatusOK, loginResponseDTO{Token: token})
}

const authUserKey = "authenticatedUserID"

func requireSession(lookup func(context.Context, string) (string, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			scheme, token, ok := strings.Cut(c.Request().Header.Get("Authorization"), " ")
			if ok && strings.EqualFold(scheme, "Bearer") && token != "" {
				id, err := lookup(c.Request().Context(), token)
				if err == nil && id != "" {
					c.Set(authUserKey, id)
					return next(c)
				}
				if err != nil && !errors.Is(err, auth.ErrUnauthenticated) {
					return fmt.Errorf("authenticate request: %w", err)
				}
			}
			c.Response().Header().Set("WWW-Authenticate", "Bearer")
			return echo.NewHTTPError(http.StatusUnauthorized, auth.ErrUnauthenticated.Error())
		}
	}
}

func authenticatedUser(c *echo.Context) string {
	id, _ := c.Get(authUserKey).(string)
	return id
}
