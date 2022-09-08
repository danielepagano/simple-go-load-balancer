package security

import (
	"fmt"
	"strings"
)

type Authorizer interface {
	AuthorizeClient(clientId string, appId string) error
}

type ClientID string
type AppID string
type ClientPermissions map[ClientID]map[AppID]struct{}

func NewAuthorizer(permissions ClientPermissions) Authorizer {
	return &simpleAuthZ{ClientPermissions: permissions}
}

type simpleAuthZ struct {
	ClientPermissions ClientPermissions
}

func (a *simpleAuthZ) AuthorizeClient(clientId string, appId string) error {
	// Let's normalize client ids to lowercase, since they are not case-sensitive
	// This is to match with https://www.rfc-editor.org/rfc/rfc5280#section-4.2.1.6
	// since we get client ids from X509 certificates
	allowedApps, found := a.ClientPermissions[ClientID(strings.ToLower(clientId))]
	if !found {
		// Client could use false as normal not found, or raise an issue if we
		// expect all incoming clients to be configured (or for better logging)
		return fmt.Errorf("client not configured: %s", clientId)
	}
	_, allowed := allowedApps[AppID(strings.ToLower(appId))]
	if !allowed {
		return fmt.Errorf("client %s not allowed to access appId %s", clientId, appId)
	}
	return nil
}
