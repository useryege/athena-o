package logout

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/operationlog/event"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	"github.com/useryege/athena/internal/walletsecret"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
	session "github.com/useryege/athena/util/session"
	settings "github.com/useryege/athena/util/settings"
)

type Handler struct {
	settingsMgr           *settings.SettingsManager
	rootPath              string
	parseToken            func(tokenString string) (jwt.Claims, accountcredentials.ApplicationRealm, error)
	revokeToken           func(ctx context.Context, id string, expiringAt time.Duration) error
	baseHRef              string
	clearSensitiveCookies func(http.ResponseWriter)
}

// NewHandler creates handler serving to do api/logout endpoint
func NewHandler(settingsMrg *settings.SettingsManager, sessionMgr *session.SessionManager, walletSecrets, wormCredentials *walletsecret.Manager, rootPath, baseHRef string) *Handler {
	clearSensitiveCookies := func(http.ResponseWriter) {}
	if walletSecrets != nil {
		clearSensitiveCookies = walletSecrets.ClearCookie
	}
	if wormCredentials != nil {
		clearPrevious := clearSensitiveCookies
		clearSensitiveCookies = func(w http.ResponseWriter) {
			clearPrevious(w)
			wormCredentials.ClearCookie(w)
		}
	}
	return &Handler{
		settingsMgr:           settingsMrg,
		rootPath:              rootPath,
		baseHRef:              baseHRef,
		parseToken:            sessionMgr.ParseLoginForRevocation,
		revokeToken:           sessionMgr.RevokeToken,
		clearSensitiveCookies: clearSensitiveCookies,
	}
}

// ServeHTTP clears the Athena auth cookie, revokes the local session token when possible, and redirects to Athena.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logoutCompleted := false
	cookieCleared, revocationConfirmed, noActiveSession := false, false, false
	var accountID string
	defer func() {
		if !logoutCompleted {
			return
		}
		if canonical, err := accountcredentials.CanonicalAccountID(accountID); err == nil {
			operationlogrecord.CaptureResource(r.Context(), "account", canonical)
		}
		operationlogrecord.CaptureBool(r.Context(), "cookieCleared", cookieCleared)
		operationlogrecord.CaptureBool(r.Context(), "revocationConfirmed", revocationConfirmed)
		operationlogrecord.CaptureBool(r.Context(), "noActiveSession", noActiveSession)
		if !noActiveSession && !revocationConfirmed {
			if recorder := operationlogrecord.FromContext(r.Context()); recorder != nil {
				recorder.Effect("IDENTITY_LOGOUT")
				recorder.Result(event.Partial, "SESSION_REVOCATION_UNCONFIRMED")
			}
		} else {
			operationlogrecord.Commit(r.Context(), "IDENTITY_LOGOUT")
		}
	}()
	realmValues := r.Header.Values(common.ApplicationRealmHeader)
	if len(realmValues) != 1 {
		http.Error(w, "application realm is required exactly once", http.StatusBadRequest)
		return
	}
	if _, present := r.URL.Query()[common.ApplicationRealmQueryParameter]; present {
		http.Error(w, "application realm query is not accepted", http.StatusBadRequest)
		return
	}
	realm, err := accountcredentials.ParseApplicationRealm(realmValues[0])
	if err != nil {
		http.Error(w, "application realm is required", http.StatusBadRequest)
		return
	}
	cookieName, err := httputil.RealmAuthCookieName(realm)
	if err != nil {
		http.Error(w, "application realm is invalid", http.StatusBadRequest)
		return
	}
	if realm == accountcredentials.ApplicationRealmMember {
		h.clearSensitiveCookies(w)
	}
	cookies := r.Cookies()
	for _, cookie := range cookies {
		if cookie.Name != cookieName && !strings.HasPrefix(cookie.Name, cookieName+"-") {
			continue
		}
		cookieCleared = true

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

	tokenString, err := httputil.JoinCookies(cookieName, cookies)
	if tokenString == "" || err != nil {
		noActiveSession = true
		logoutCompleted = true
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	claims, tokenRealm, err := h.parseToken(tokenString)
	if err != nil || tokenRealm != realm {
		noActiveSession = true
		logoutCompleted = true
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	mapClaims, err := jwtutil.MapClaims(claims)
	if err != nil {
		noActiveSession = true
		logoutCompleted = true
		http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
		return
	}

	id := jwtutil.StringField(mapClaims, "jti")
	accountID = jwtutil.StringField(mapClaims, "sub")
	if exp, err := jwtutil.ExpirationTime(mapClaims); err == nil && id != "" {
		if err := h.revokeToken(context.Background(), id, time.Until(exp)); err != nil {
			log.Warnf("failed to invalidate logout token: %v", err)
		} else {
			revocationConfirmed = true
		}
	} else {
		noActiveSession = true
	}

	logoutCompleted = true
	http.Redirect(w, r, logoutRedirectURL, http.StatusSeeOther)
}
