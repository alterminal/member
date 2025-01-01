package repo

import (
	authModel "github.com/alterminal/auth/model"
	"github.com/alterminal/member/model"
	"gorm.io/gorm"
)

func Init(db *gorm.DB) {
	db.AutoMigrate(&model.Organization{})
	db.AutoMigrate(&model.Role{})
	db.AutoMigrate(&model.AccountRole{})
	db.AutoMigrate(&model.Space{})
	db.AutoMigrate(&model.SubscriptionPlan{})
	db.AutoMigrate(&model.Subscription{})
	var subscriptions []model.Subscription
	db.Where("completed_at IS NULL").
		Where("canceled_at IS NULL").
		Find(&subscriptions)
	for _, subscription := range subscriptions {
		go subscription.Watch(db)
	}
}

func AccountRoles(db *gorm.DB, account authModel.Account) []model.Role {
	var roles []model.Role
	db.Raw(`SELECT * FROM roles WHERE id IN (SELECT role_id FROM account_roles WHERE account_id = ?)`, account.ID).Scan(&roles)
	return roles
}

func AccountOrganizations(db *gorm.DB, account authModel.Account) []model.Organization {
	var organizations []model.Organization
	db.Raw(`SELECT * FROM organizations 
		WHERE id IN (SELECT organization_id FROM roles WHERE id IN (SELECT role_id FROM account_roles WHERE account_id = ?))
		`, account.ID).Scan(&organizations)
	return organizations
}
