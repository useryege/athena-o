package rbacpolicy

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"

	jwtutil "github.com/useryege/athena/util/jwt"
	"github.com/useryege/athena/util/rbac"
)

// RBACPolicyEnforcer provides an RBAC Claims Enforcer which additionally consults AppProject
// roles, jwt tokens, and groups. It is backed by a AppProject informer/lister cache and does not
// make any API calls during enforcement.
type RBACPolicyEnforcer struct {
	enf *rbac.Enforcer
	// projLister applister.AppProjectNamespaceLister
	scopes []string
}

// NewRBACPolicyEnforcer returns a new RBAC Enforcer for the Athena API Server
func NewRBACPolicyEnforcer(enf *rbac.Enforcer) *RBACPolicyEnforcer {
	return &RBACPolicyEnforcer{
		enf: enf,
		// projLister: projLister,
		scopes: nil,
	}
}

func (p *RBACPolicyEnforcer) SetScopes(scopes []string) {
	p.scopes = scopes
}

func (p *RBACPolicyEnforcer) GetScopes() []string {
	scopes := p.scopes
	if scopes == nil {
		scopes = rbac.DefaultScopes
	}
	return scopes
}

// func IsProjectSubject(subject string) bool {
// 	_, _, ok := GetProjectRoleFromSubject(subject)
// 	return ok
// }

func GetProjectRoleFromSubject(subject string) (string, string, bool) {
	parts := strings.Split(subject, ":")
	if len(parts) == 3 && parts[0] == "proj" {
		return parts[1], parts[2], true
	}
	return "", "", false
}

// EnforceClaims is an RBAC claims enforcer specific to the Athena API server
func (p *RBACPolicyEnforcer) EnforceClaims(claims jwt.Claims, rvals ...any) bool {
	mapClaims, err := jwtutil.MapClaims(claims)
	if err != nil {
		return false
	}

	subject := jwtutil.GetUserIdentifier(mapClaims)
	// Web application mode does not use AppProject-scoped RBAC, so we skip
	// project token/group policy resolution here.
	var runtimePolicy string
	var projName string

	// NOTE: This calls prevent multiple creation of the wrapped enforcer
	enforcer := p.enf.CreateEnforcerWithRuntimePolicy(projName, runtimePolicy)

	// Check the subject. This is typically the 'admin' case.
	// NOTE: the call to EnforceWithCustomEnforcer will also consider the default role
	vals := append([]any{subject}, rvals[1:]...)
	if p.enf.EnforceWithCustomEnforcer(enforcer, vals...) {
		return true
	}

	scopes := p.scopes
	if scopes == nil {
		scopes = rbac.DefaultScopes
	}
	// Finally check if any of the user's groups grant them permissions
	groups := jwtutil.GetScopeValues(mapClaims, scopes)

	// Get groups to reduce the amount to checking groups
	groupingPolicies, err := enforcer.GetGroupingPolicy()
	if err != nil {
		log.WithError(err).Error("failed to get grouping policy")
		return false
	}
	for gidx := range groups {
		for gpidx := range groupingPolicies {
			// Prefilter user groups by groups defined in the model
			if groupingPolicies[gpidx][0] == groups[gidx] {
				vals := append([]any{groups[gidx]}, rvals[1:]...)
				if p.enf.EnforceWithCustomEnforcer(enforcer, vals...) {
					return true
				}
				break
			}
		}
	}
	logCtx := log.WithFields(log.Fields{"claims": claims, "rval": rvals, "subject": subject, "groups": groups, "project": projName, "scopes": scopes})
	logCtx.Debug("enforce failed")
	return false
}

// Project-scoped RBAC is intentionally disabled for the current web application.
