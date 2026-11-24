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

type registerRequestDTO struct {
	Username string `json:"username" binding:"required" example:"alex"`
	Email    string `json:"email" binding:"required" format:"email" example:"alex@example.com"`
	Password string `json:"password" binding:"required" format:"password" minLength:"8"`
}

type forgotPasswordRequestDTO struct {
	Email string `json:"email" binding:"required" format:"email" example:"alex@example.com"`
}

type resetPasswordRequestDTO struct {
	Email    string `json:"email" binding:"required" format:"email" example:"alex@example.com"`
	Code     string `json:"code" binding:"required" example:"K3QZ7WMA"`
	Password string `json:"password" binding:"required" format:"password" minLength:"8"`
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
	e.POST("/api/v2/register", h.Register, limit, middleware.BodyLimit(8*1024))
	e.POST("/api/v2/forgot-password", h.ForgotPassword, limit, middleware.BodyLimit(8*1024))
	e.POST("/api/v2/reset-password", h.ResetPassword, limit, middleware.BodyLimit(8*1024))
}

// ForgotPassword issues a password reset code.
// @Summary Request a password reset code
// @Description Create a one-time reset code for the account with this email, replacing any earlier code. The response is the same whether or not the account exists. Codes expire after 15 minutes. Email delivery is not implemented yet: the API writes the code to its log. Use the code with POST /reset-password.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body forgotPasswordRequestDTO true "Account email"
// @Success 202 "Reset code issued if the account exists"
// @Failure 400 {object} errorResponseDTO "Invalid request body or email"
// @Failure 413 {object} errorResponseDTO "Request body too large"
// @Failure 429 {object} errorResponseDTO "Too many attempts"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *echo.Context) error {
	var req forgotPasswordRequestDTO
	if err := bindAuthRequest(c, &req); err != nil {
		return err
	}
	err := h.service.ForgotPassword(c.Request().Context(), req.Email)
	var validationErr *auth.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return echo.NewHTTPError(http.StatusBadRequest, validationErr.Error())
	case err != nil:
		return fmt.Errorf("forgot password: %w", err)
	}
	return c.NoContent(http.StatusAccepted)
}

// ResetPassword sets a new password using a reset code.
// @Summary Reset password
// @Description Choose a new password with the email and a code from POST /forgot-password. Passwords need at least 8 characters and at most 1024 bytes. The code works once, and resetting signs out the account's current session. Sign in with POST /login afterwards.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body resetPasswordRequestDTO true "Email, reset code, and new password"
// @Success 204 "Password changed"
// @Failure 400 {object} errorResponseDTO "Invalid request body, field, or reset code"
// @Failure 413 {object} errorResponseDTO "Request body too large"
// @Failure 429 {object} errorResponseDTO "Too many attempts"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /reset-password [post]
func (h *AuthHandler) ResetPassword(c *echo.Context) error {
	var req resetPasswordRequestDTO
	if err := bindAuthRequest(c, &req); err != nil {
		return err
	}
	err := h.service.ResetPassword(c.Request().Context(), req.Email, req.Code, req.Password)
	var validationErr *auth.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return echo.NewHTTPError(http.StatusBadRequest, validationErr.Error())
	case errors.Is(err, auth.ErrResetCode):
		return echo.NewHTTPError(http.StatusBadRequest, auth.ErrResetCode.Error())
	case errors.Is(err, auth.ErrBusy):
		return echo.NewHTTPError(http.StatusTooManyRequests, auth.ErrBusy.Error())
	case err != nil:
		return fmt.Errorf("reset password: %w", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func bindAuthRequest(c *echo.Context, req any) error {
	if err := c.Bind(req); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) && httpErr.Code == http.StatusRequestEntityTooLarge {
			return err
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	return nil
}

// Register creates an account.
// @Summary Register
// @Description Create an account with a username, email, and password. The username and email are trimmed, and the email is lowercased. Passwords need at least 8 characters and at most 1024 bytes. Returns an empty body; sign in with POST /login to get a session token.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param account body registerRequestDTO true "Account details"
// @Success 201 "Account created"
// @Failure 400 {object} errorResponseDTO "Invalid request body or field"
// @Failure 409 {object} errorResponseDTO "Username or email already in use"
// @Failure 413 {object} errorResponseDTO "Request body too large"
// @Failure 429 {object} errorResponseDTO "Too many attempts"
// @Failure 500 {object} errorResponseDTO "Unexpected server error"
// @Router /register [post]
func (h *AuthHandler) Register(c *echo.Context) error {
	var req registerRequestDTO
	if err := c.Bind(&req); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) && httpErr.Code == http.StatusRequestEntityTooLarge {
			return err
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	err := h.service.Register(c.Request().Context(), auth.Registration{
		Username: req.Username, Email: req.Email, Password: req.Password,
	})
	var validationErr *auth.ValidationError
	switch {
	case errors.As(err, &validationErr):
		return echo.NewHTTPError(http.StatusBadRequest, validationErr.Error())
	case errors.Is(err, auth.ErrUsernameTaken), errors.Is(err, auth.ErrEmailTaken):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrBusy):
		return echo.NewHTTPError(http.StatusTooManyRequests, auth.ErrBusy.Error())
	case err != nil:
		return fmt.Errorf("register: %w", err)
	}
	return c.NoContent(http.StatusCreated)
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
