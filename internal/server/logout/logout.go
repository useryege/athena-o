package logout

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
	session "github.com/useryege/athena/util/session"
	settings "github.com/useryege/athena/util/settings"
)

var (
	tokenPattern             = regexp.MustCompile(`{{token}}`)
	logoutRedirectURLPattern = regexp.MustCompile(`{{logoutRedirectURL}}`)
)

type Handler struct {
	settingsMgr *settings.SettingsManager
	rootPath    string
	verifyToken func(ctx context.Context, tokenString string) (jwt.Claims, string, error)
	revokeToken func(ctx context.Context, id string, expiringAt time.Duration) error
	baseHRef    string
}

// NewHandler creates handler serving to do api/logout endpoint
func NewHandler(settingsMrg *settings.SettingsManager, sessionMgr *session.SessionManager, rootPath, baseHRef string) *Handler {
	return &Handler{
		settingsMgr: settingsMrg,
		rootPath:    rootPath,
		baseHRef:    baseHRef,
		verifyToken: sessionMgr.VerifyToken,
		revokeToken: sessionMgr.RevokeToken,
	}
}

func constructLogoutURL(logoutURL, token, logoutRedirectURL string) string {
	constructedLogoutURL := tokenPattern.ReplaceAllString(logoutURL, token)
	return logoutRedirectURLPattern.ReplaceAllString(constructedLogoutURL, logoutRedirectURL)
}

// ServeHTTP is the logout handler for Athena and constructs OIDC logout URL and redirects to it for OIDC issued sessions,
// and redirects user to '/login' for athena issued sessions
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tokenString string
	var oidcConfig *settings.OIDCConfig

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

	cookies := r.Cookies()
	tokenString, err = httputil.JoinCookies(common.AuthCookieName, cookies)
	if tokenString == "" || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		http.Error(w, "Failed to retrieve Athena auth token: "+err.Error(), http.StatusBadRequest)
		return
	}

	for _, cookie := range cookies {
		if !strings.HasPrefix(cookie.Name, common.AuthCookieName) {
			continue
		}

		athenaCookie := http.Cookie{
			Name:  cookie.Name,
			Value: "",
		}

		athenaCookie.Path = "/" + strings.TrimRight(strings.TrimLeft(h.baseHRef, "/"), "/")
		w.Header().Add("Set-Cookie", athenaCookie.String())
	}

	claims, _, err := h.verifyToken(r.Context(), tokenString)
	if err != nil {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	mapClaims, err := jwtutil.MapClaims(claims)
	if err != nil {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	issuer := jwtutil.StringField(mapClaims, "iss")
	id := jwtutil.StringField(mapClaims, "jti")
	if exp, err := jwtutil.ExpirationTime(mapClaims); err == nil && id != "" {
		if err := h.revokeToken(context.Background(), id, time.Until(exp)); err != nil {
			log.Warnf("failed to invalidate token '%s': %v", id, err)
		}
	}

	if athenaSettings.OIDCConfig() == nil || athenaSettings.OIDCConfig().LogoutURL == "" || issuer == session.SessionManagerClaimsIssuer {
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
	} else {
		oidcConfig = athenaSettings.OIDCConfig()
		logoutURL := constructLogoutURL(oidcConfig.LogoutURL, tokenString, logoutRedirectURL)
		http.Redirect(w, r, logoutURL, http.StatusSeeOther)
	}
}
