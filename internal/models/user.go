package models

type AccessGrant struct {
	UserDN           string
	GroupDN          string // The specific group to revoke
	AccessExpiryTime int64
	RawAttribute     string // The exact string (e.g., "cn=Group...|12345")
}
