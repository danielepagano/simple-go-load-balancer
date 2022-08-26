package lbproxy

// RateLimitManager tracks rate limits in an arbitrary scope
// Each method is a request for an action against that scope,
// and most will return true if the request is allowed, false if denied
//
// In a more complex system, the response could be struct that allows for
// in-between states, for example data transfers may be delayed to maintain a
// goal data rate across the entire scope (e.g. bandwidth limit across all connections of one client)
type RateLimitManager interface {
	addConnection() bool
	releaseConnection()

	// Extension example: interface to track bandwidth usage and rate-limit it
	dataTransfer(bytesTransferred int64, isResponse bool) bool
}

// TODO: implement interface
