package internal

import (
	"fmt"
	"strings"
)

type AuthorizationProvider interface {
	AuthorizeClient(clientId string, appId string) (bool, error)
}

type SimpleAuthZ struct {
	ClientPermissions map[string][]string
}

func (a *SimpleAuthZ) AuthorizeClient(clientId string, appId string) (bool, error) {
	// Let's normalize client ids to lowercase, since they are not case-sensitive
	// This is to match with https://www.rfc-editor.org/rfc/rfc5280#section-4.2.1.6
	// since we get client ids from X509 certificates
	allowedApps, found := a.ClientPermissions[strings.ToLower(clientId)]
	if !found {
		// Client could use false as normal not found, or raise an issue if we
		// expect all incoming clients to be configured (or for better logging)
		return false, fmt.Errorf("client not configured:" + clientId)
	}
	return contains(allowedApps, appId), nil
}

// Apps are not guaranteed to be sorted, so perform a linear search
func contains(apps []string, id string) bool {
	for _, a := range apps {
		// Let's make app ids also not case-sensitive, since the client ids are not
		if strings.EqualFold(a, id) {
			return true
		}
	}
	return false
}
