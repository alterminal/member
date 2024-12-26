package model

import (
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Space struct {
	ID             string  `json:"id" gorm:"type:char(19);primaryKey"`
	OrganizationID string  `json:"organizationId" gorm:"type:char(19);index"`
	ParentId       *string `json:"parentId" gorm:"type:char(19);index"`
	Name           string  `json:"name" gorm:"type:varchar(255)"`
}

func (a *Space) BeforeSave(tx *gorm.DB) error {
	if a.ParentId != nil {
		parenct := Space{}
		err := tx.First(&parenct, "id = ?", a.ParentId).Error
		if err != nil {
			a.ParentId = nil
		}
		if parenct.OrganizationID != a.OrganizationID {
			a.ParentId = nil
		}
	}
	return nil
}

func (a *Space) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

func (s *Space) Children(tx *gorm.DB) []Space {
	var spaces []Space
	tx.Where("parent_id = ?", s.ID).Find(&spaces)
	return spaces
}

func (s *Space) CreateSubscriptionPlan(tx *gorm.DB, currency string, price int) (*SubscriptionPlan, error) {
	subscriptionPlan := SubscriptionPlan{
		SpaceID:  s.ID,
		Currency: currency,
		Price:    price,
	}
	err := tx.Create(&subscriptionPlan).Error
	if err != nil {
		return nil, err
	}
	return &subscriptionPlan, nil
}

func (s *Space) SubscriptionPlans(tx *gorm.DB) []*SubscriptionPlan {
	var subscriptionPlans []*SubscriptionPlan = make([]*SubscriptionPlan, 0)
	tx.Find(&subscriptionPlans, "space_id = ?", s.ID)
	return subscriptionPlans
}

type SubscriptionPlan struct {
	ID             string `json:"id" gorm:"type:char(19);primaryKey"`
	SpaceID        string `json:"spaceId" gorm:"type:char(19);index"`
	Currency       string `json:"currency" gorm:"type:varchar(3)"`
	Price          int    `json:"price" gorm:"type:int"`
	PaymentGateway string `json:"paymentGateway" gorm:"type:varchar(255)"`
}

func (a *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

type Subscription struct {
	ID                 string `json:"id" gorm:"type:char(19);primaryKey"`
	SubscriptionPlanId string `json:"subscriptionPlanId" gorm:"type:char(19);index"`
	ConsumerId         string `json:"ConsumerId" gorm:"type:char(19);index"`
}

func (a *Subscription) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}
