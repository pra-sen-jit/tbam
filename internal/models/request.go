package models

type UserRequest struct {
	UID             string `json:"uid"`
	GroupAssociated string `json:"grp_associated"` // The base group they belong to
	PrivilegeAccess string `json:"privilege_access"` // The target group to grant
	EndDate         string `json:"end_date"`       // Format: "2026-05-04"
	EndTime         string `json:"end_time"`       // Format: "15:30:00"
}
