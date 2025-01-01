package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/alterminal/member/payment"
	"github.com/bwmarrin/snowflake"
	"github.com/spf13/viper"

	"gorm.io/gorm"
)

const (
	Watch_interval = 5 * time.Second
)

type Space struct {
	ID             string     `json:"id" gorm:"type:char(19);primaryKey"`
	OrganizationID string     `json:"organizationId" gorm:"type:char(19);index"`
	ParentId       *string    `json:"parentId" gorm:"type:char(19);index"`
	Name           string     `json:"name" gorm:"type:varchar(255)"`
	DisabledAt     *time.Time `json:"disabledAt" gorm:"type:datetime"`
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

func (s *Space) CreateSubscriptionPlan(tx *gorm.DB, planName, paymentGateway string, currency string, price int) (*SubscriptionPlan, error) {
	subscriptionPlan := SubscriptionPlan{
		PlanName:       planName,
		PaymentGateway: paymentGateway,
		SpaceID:        s.ID,
		Currency:       currency,
		Price:          price,
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
	PlanName       string `json:"planName" gorm:"type:varchar(255)"`
	SpaceID        string `json:"spaceId" gorm:"type:char(19);index"`
	Currency       string `json:"currency" gorm:"type:char(3)"`
	Price          int    `json:"price" gorm:"type:int"`
	PaymentGateway string `json:"paymentGateway" gorm:"type:varchar(255)"`
}

func (a *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

func (a *SubscriptionPlan) CreateSubscription(db *gorm.DB) (*payment.Subscription, error) {
	if len(a.GetSubscriptions(db)) > 0 {
		return nil, fmt.Errorf("subscription already exists")
	}
	var err error
	var subscriptions Subscription
	err = db.Where("completed_at IS NULL").
		Where("canceled_at IS NULL").
		Where("subscription_plan_id = ?", a.ID).First(&subscriptions).Error
	var sub *payment.Subscription
	paymentGateway := a.GetPaymentGateway()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		sub, err = paymentGateway.CreateSubscription(a.PlanName, a.Price, a.Currency)
	} else {
		sub, err = paymentGateway.RetrieveSubscription(subscriptions.PaymentId)
	}
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	subscription := Subscription{
		SubscriptionPlanId: a.ID,
		Secret:             sub.ID,
		PaymentId:          sub.ID,
	}
	db.Create(&subscription)
	go subscription.Watch(db)
	return sub, err
}

func (a *SubscriptionPlan) GetPaymentGateway() payment.PaymentGateway {
	return getPaymentGateway(a.PaymentGateway)
}

func (a *SubscriptionPlan) GetSubscriptions(tx *gorm.DB) []*Subscription {
	var subscriptions []*Subscription = make([]*Subscription, 0)
	tx.Where("completed_at IS NOT NULL").Where("canceled_at IS NULL").Where("subscription_plan_id = ?", a.ID).Find(&subscriptions)
	return subscriptions
}

type Subscription struct {
	ID                 string `json:"id" gorm:"type:char(19);primaryKey"`
	SubscriptionPlanId string `json:"subscriptionPlanId" gorm:"type:char(19);index"`
	PaymentId          string `json:"paymentId" gorm:"type:varchar(128);"`
	Secret             string `json:"secret" gorm:"type:varchar(255)"`
	CreatedAt          time.Time
	CompletedAt        *time.Time
	CanceledAt         *time.Time
}

func (a *Subscription) GetSubscriptionPlan(tx *gorm.DB) *SubscriptionPlan {
	subscriptionPlan := SubscriptionPlan{}
	tx.First(&subscriptionPlan, "id = ?", a.SubscriptionPlanId)
	return &subscriptionPlan
}

func (a *Subscription) BeforeCreate(tx *gorm.DB) error {
	node, _ := snowflake.NewNode(0)
	a.ID = node.Generate().String()
	return nil
}

func (a *Subscription) Watch(tx *gorm.DB) {
	paymentGateway := a.GetSubscriptionPlan(tx).GetPaymentGateway()
	if a.CanceledAt != nil {
		return
	}
	for {
		sub, err := paymentGateway.RetrieveSubscription(a.PaymentId)
		if err != nil {
			return
		}
		now := time.Now()
		if a.CompletedAt == nil && sub.Completed {
			a.Complete(tx)
		}
		if sub.Canceled {
			a.CanceledAt = &now
			tx.Save(a)
			return
		}
		time.Sleep(Watch_interval)
	}
}

func (a *Subscription) Complete(tx *gorm.DB) error {
	// TODO: conflict check and lock
	now := time.Now()
	a.CompletedAt = &now
	if len(a.GetSubscriptionPlan(tx).GetSubscriptions(tx)) > 0 {
		fmt.Println("subscription already exists")
		// a.Cancel(tx)
		return fmt.Errorf("subscription already exists")
	}
	tx.Save(a)
	return nil
}

func (a *Subscription) Cancel(tx *gorm.DB) error {
	if a.CompletedAt == nil {
		return fmt.Errorf("subscription not completed")
	}
	a.GetSubscriptionPlan(tx).GetPaymentGateway().CancelSubscription(a.PaymentId)
	a.CanceledAt = FNow()
	tx.Save(a)
	return nil
}

func getPaymentGateway(paymentGateway string) payment.PaymentGateway {
	switch paymentGateway {
	case "stripe":
		return &payment.Stripe{
			Key: viper.GetString("stripe.key"),
		}
	}
	return nil
}

func FNow() *time.Time {
	now := time.Now()
	return &now
}
