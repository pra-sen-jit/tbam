package models

type UserRequest struct {
	UID             string `json:"uid"`
	GroupAssociated string `json:"grp_associated"`
	PrivilegeAccess string `json:"privilege_access"`
	StartDate       string `json:"start_date"`
	StartTime       string `json:"start_time"`
	EndDate         string `json:"end_date"`
	EndTime         string `json:"end_time"`
}
