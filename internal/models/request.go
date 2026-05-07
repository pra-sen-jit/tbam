package models

type UserRequest struct {
	UID             string `json:"uid"`
	GroupAssociated string `json:"grp_associated"`
	PrivilegeAccess string `json:"privilege_access"`
	EndDate         string `json:"end_date"`  // Format: "2026-05-04"
	EndTime         string `json:"end_time"`  // Format: "15:30:00"
}
