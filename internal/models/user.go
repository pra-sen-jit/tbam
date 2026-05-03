package models

// ExpiringAccess represents an identity that has time-bound access scheduled for revocation.
type ExpiringAccess struct {
	DN               string // Distinguished Name
	AccessExpiryTime int64  // Unix timestamp of when the access expires
}
