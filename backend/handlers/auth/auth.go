package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/subbu/family_tree/config"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	authservice "github.com/subbu/family_tree/services/auth"
)

type Handler struct {
	auth authservice.Service
	cfg  config.Config
}

func NewHandler(auth authservice.Service, cfg config.Config) *Handler {
	return &Handler{auth: auth, cfg: cfg}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]bool{
		"google_enabled": h.cfg.GoogleClientID != "" && h.cfg.GoogleClientSecret != "",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GoogleClientID == "" || h.cfg.GoogleClientSecret == "" {
		response.Error(w, http.StatusServiceUnavailable, "Google OAuth is not configured. Set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in .env")
		return
	}

	state := randomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, h.auth.LoginURL(state), http.StatusFound)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_state", http.StatusFound)
		return
	}

	token, _, err := h.auth.HandleCallback(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, h.cfg.FrontendURL+"/login?error=oauth_failed", http.StatusFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	redirectURL := h.cfg.FrontendURL + "/?token=" + url.QueryEscape(token)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	response.JSON(w, http.StatusOK, user)
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handler) Exchange(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, user, err := h.auth.HandleCallback(r.Context(), payload.Code)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

func randomState() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}