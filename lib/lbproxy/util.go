package lbproxy

import "strings"

const closedConnError = "use of closed network connection"

func IsErrorClosedNetworkConnection(err error) bool {
	return err != nil && strings.Contains(err.Error(), closedConnError)
}
