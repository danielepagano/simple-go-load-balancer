package lbproxy

// RateLimitManager tracks rate limits in an arbitrary scope
// Each method is a request for an action against that scope,
// and most will return true if the request is allowed, false if denied
//
// In a more complex system, the response could be struct that allows for
// in-between states, for example data transfers may be delayed to maintain a
// goal data rate across the entire scope (e.g. bandwidth limit across all connections of one client)
type RateLimitManager interface {
	// AddConnection checks that the quantity and timing of a connection request matches the policy for this
	// scope, and returns true if so, and false otherwise
	AddConnection() bool

	// ReleaseConnection decreases the count of active connections to support max open connections capping
	ReleaseConnection()
}

// TODO: implement interface
