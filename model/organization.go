package model

import (
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Organization struct {
	ID   string `json:"id" gorm:"type:char(19);primaryKey"`
	Name string `json:"name" gorm:"type:varchar(255)"`
}

func (a *Organization) BeforeDelete(tx *gorm.DB) error {
	tx.Exec("DELETE FROM account_roles WHERE role_id IN (SELECT id FROM roles WHERE organization_id = ?)", a.ID)
	tx.Exec("DELETE FROM roles WHERE organization_id = ?", a.ID)
	return nil
}

func (a *Organization) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

type Role struct {
	ID             string `json:"id" gorm:"type:char(19);primaryKey"`
	OrganizationID string `json:"organizationId" gorm:"type:char(19);index"`
	Name           string `json:"name" gorm:"type:varchar(255)"`
}

func (a *Role) BeforeDelete(tx *gorm.DB) error {
	tx.Exec("DELETE FROM account_roles WHERE role_id = ?", a.ID)
	return nil
}

func (a *Role) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

type AccountRole struct {
	AccountID string `json:"accountId" gorm:"type:char(19);primaryKey"`
	RoleID    string `json:"roleId" gorm:"type:char(19);primaryKey"`
}
