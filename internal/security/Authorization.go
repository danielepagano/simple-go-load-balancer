package security

import (
	"fmt"
	"strings"
)

type AuthorizationProvider interface {
	AuthorizeClient(clientId string, appId string) (bool, error)
}

func NewAuthorizationProvider(config ServerSecurityConfig, permissions map[string]map[string]struct{}) AuthorizationProvider {
	if config.EnableMutualTLS {
		return &simpleAuthZ{ClientPermissions: permissions}
	}
	return &noOpAuthZ{}
}

type simpleAuthZ struct {
	ClientPermissions map[string]map[string]struct{}
}

func (a *simpleAuthZ) AuthorizeClient(clientId string, appId string) (bool, error) {
	// Let's normalize client ids to lowercase, since they are not case-sensitive
	// This is to match with https://www.rfc-editor.org/rfc/rfc5280#section-4.2.1.6
	// since we get client ids from X509 certificates
	allowedApps, found := a.ClientPermissions[strings.ToLower(clientId)]
	if !found {
		// Client could use false as normal not found, or raise an issue if we
		// expect all incoming clients to be configured (or for better logging)
		return false, fmt.Errorf("client not configured: %s", clientId)
	}
	_, allowed := allowedApps[strings.ToLower(appId)]
	return allowed, nil
}
