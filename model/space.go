package model

import "gorm.io/gorm"

type Space struct {
	ID             string `json:"id" gorm:"type:char(19);primaryKey"`
	OrganizationID string `json:"-" gorm:"type:char(19);index"`
	ParenctID      string `json:"parenctId" gorm:"type:char(19);index"`
	Name           string `json:"name" gorm:"type:varchar(255)"`
}

func (s *Space) Children(tx *gorm.DB) []Space {
	var spaces []Space
	tx.Where("parenct_id = ?", s.ID).Find(&spaces)
	return spaces
}
