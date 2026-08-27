package logout

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/walletsecret"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
	session "github.com/useryege/athena/util/session"
	settings "github.com/useryege/athena/util/settings"
)

type Handler struct {
	settingsMgr       *settings.SettingsManager
	rootPath          string
	parseToken        func(tokenString string) (jwt.Claims, error)
	revokeToken       func(ctx context.Context, id string, expiringAt time.Duration) error
	baseHRef          string
	clearWalletSecret func(http.ResponseWriter)
}

// NewHandler creates handler serving to do api/logout endpoint
func NewHandler(settingsMrg *settings.SettingsManager, sessionMgr *session.SessionManager, walletSecrets *walletsecret.Manager, rootPath, baseHRef string) *Handler {
	clearWalletSecret := func(http.ResponseWriter) {}
	if walletSecrets != nil {
		clearWalletSecret = walletSecrets.ClearCookie
	}
	return &Handler{
		settingsMgr:       settingsMrg,
		rootPath:          rootPath,
		baseHRef:          baseHRef,
		parseToken:        sessionMgr.ParseLoginForRevocation,
		revokeToken:       sessionMgr.RevokeToken,
		clearWalletSecret: clearWalletSecret,
	}
}

// ServeHTTP clears the Athena auth cookie, revokes the local session token when possible, and redirects to Athena.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.clearWalletSecret(w)
	cookies := r.Cookies()
	for _, cookie := range cookies {
		if !strings.HasPrefix(cookie.Name, common.AuthCookieName) {
			continue
		}

		athenaCookie := http.Cookie{
			Name:     cookie.Name,
			Value:    "",
			MaxAge:   -1,
			Expires:  time.Unix(1, 0),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}

		athenaCookie.Path = "/" + strings.TrimRight(strings.TrimLeft(h.baseHRef, "/"), "/")
		w.Header().Add("Set-Cookie", athenaCookie.String())
	}

	athenaSettings, err := h.settingsMgr.GetSettings()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		http.Error(w, "Failed to retrieve Athena settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	athenaURL, err := athenaSettings.AthenaURLForRequest(r)
	if err != nil {
		log.Warnf("unable to find Athena URL from config: %v", err)
	}
	if athenaURL == "" {
		// golang does not provide any easy way to determine scheme of current request
		// so redirecting ot http which will auto-redirect too https if necessary
		host := strings.TrimRight(r.Host, "/")
		athenaURL = "http://" + host + "/" + strings.TrimRight(strings.TrimLeft(h.rootPath, "/"), "/")
	}

	logoutRedirectURL := strings.TrimRight(strings.TrimLeft(athenaURL, "/"), "/")

	tokenString, err := httputil.JoinCookies(common.AuthCookieName, cookies)
	if tokenString == "" || err != nil {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	claims, err := h.parseToken(tokenString)
	if err != nil {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	mapClaims, err := jwtutil.MapClaims(claims)
	if err != nil {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	id := jwtutil.StringField(mapClaims, "jti")
	if exp, err := jwtutil.ExpirationTime(mapClaims); err == nil && id != "" {
		if err := h.revokeToken(context.Background(), id, time.Until(exp)); err != nil {
			log.Warnf("failed to invalidate logout token: %v", err)
		}
	}

	http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
}
