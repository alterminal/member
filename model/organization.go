package model

type Organization struct {
	ID   string `json:"id" gorm:"type:char(19);primaryKey"`
	Name string `json:"name" gorm:"type:varchar(255)"`
}

type Role struct {
	ID             string `json:"id" gorm:"type:char(19);primaryKey"`
	OrganizationID string `json:"-" gorm:"type:char(19);index"`
	Name           string `json:"name" gorm:"type:varchar(255)"`
}

type AccountRole struct {
	AccountID string `json:"accountId" gorm:"type:char(19);primaryKey"`
	RoleID    string `json:"roleId" gorm:"type:char(19);primaryKey"`
}
