package api

import (
	"errors"
	"strconv"
	"strings"

	authApi "github.com/alterminal/auth/api"
	auth "github.com/alterminal/auth/model"
	"github.com/alterminal/auth/sdk"
	"github.com/alterminal/common/mid"
	"github.com/alterminal/member/model"
	"github.com/alterminal/member/repo"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAccount(authClient sdk.Client) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		splited := strings.Split(token, " ")
		if len(splited) != 2 {
			return
		}
		account, err := authClient.Retrieve(splited[1])
		if err != nil {
			return
		}
		ctx.Set("account", account)
	}
}

func IsAdmin(ctx *gin.Context) {
	accountInterface, exists := ctx.Get("account")
	if !exists {
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
		return
	}
	account := accountInterface.(auth.Account)
	if account.Namespace != "admin" {
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
		return
	}
}

func IsTenant(ctx *gin.Context) {
	accountInterface, exists := ctx.Get("account")
	if !exists {
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
		return
	}
	account := accountInterface.(auth.Account)
	if account.Namespace != "tenant" {
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
		return
	}
}

func IsAdminOfOrganization(db *gorm.DB) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		accountInterface, exists := ctx.Get("account")
		if !exists {
			ctx.JSON(401, gin.H{"error": "unauthorized"})
			ctx.Abort()
			return
		}
		account := accountInterface.(auth.Account)
		if account.Namespace != "tenant" {
			ctx.JSON(401, gin.H{"error": "unauthorized"})
			ctx.Abort()
			return
		}
		organizationId := ctx.Param("id")
		if organizationId == "" {
			return
		}
		var organization model.Organization
		err := db.First(&organization, organizationId).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.JSON(404, gin.H{"error": "organization not found"})
				ctx.Abort()
				return
			}
			ctx.JSON(404, gin.H{"error": "organization not found"})
			ctx.Abort()
			return
		}
		repo.AccountRoles(db, account)
		ctx.Set("organization", organization)
		for _, role := range repo.AccountRoles(db, account) {
			if role.OrganizationID == organizationId && role.Name == "admin" {
				return
			}
		}
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
	}
}

func IsSpaceAdmin(db *gorm.DB) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		var space model.Space
		err := db.First(&space, id).Error
		if err != nil {
			ctx.Abort()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				ctx.JSON(404, gin.H{"error": "space not found"})
				return
			}
			ctx.JSON(500, gin.H{"error": "internal server error"})
			return
		}
		roles := repo.AccountRoles(db, ctx.MustGet("account").(auth.Account))
		for _, role := range roles {
			if role.OrganizationID == space.OrganizationID {
				ctx.Set("space", space)
				return
			}
		}
		ctx.JSON(401, gin.H{"error": "unauthorized"})
		ctx.Abort()
	}
}

func Run(db *gorm.DB, authClient sdk.Client) {
	router := gin.Default()
	router.Use(mid.AccessControllAllowfunc(mid.AccessControllAllowConfig{
		Origin:  "*",
		Headers: "*",
		Methods: "*",
	}))
	api := &Api{db: db, authClient: authClient}
	router.Use(GetAccount(authClient))

	router.POST("/tenants", IsAdmin, api.CreateTenant)
	router.GET("/tenants", IsAdmin, api.ListTenants)
	router.DELETE("/tenants/:id", IsAdmin, api.DeleteTenant)
	router.GET("/tenants/:id", IsAdmin, api.GetAccount)
	router.PUT("/tenants/:id/password", IsAdmin, api.SetPassword)

	router.POST("/organizations", IsAdmin, api.CreateOrganization)
	// router.DELETE("/organizations/:id", IsAdmin, api.DeleteOrganization)
	router.GET("/organizations/all", IsAdmin, api.ListAllOrganizations)

	router.DELETE("/organizations/roles/:id", IsAdmin, api.DeleteRole)
	router.POST("/organizations/:id/roles", IsAdmin, api.CreateRole)
	router.GET("/organizations/:id/roles", IsAdmin, api.ListRole)
	router.POST("/organizations/roles/:id/account", IsAdmin, api.SetAccountRole)
	router.DELETE("/organizations/roles/:id/account", IsAdmin, api.SetAccountRole)

	router.GET("/organizations", IsTenant, api.ListMyOrganizations)
	router.POST("/organizations/:id/spaces", IsAdminOfOrganization(db), api.CreateSpace)
	router.GET("/organizations/:id/spaces", IsAdminOfOrganization(db), api.ListSpaces)
	router.POST("/organizations/:id/consumers", IsAdminOfOrganization(db), api.CreateConsumer)
	router.GET("/organizations/:id/consumers", IsAdminOfOrganization(db), api.ListConsumer)
	// router.DELETE("/spaces/:id", IsSpaceAdmin(db), api.DeleteSpace)
	router.GET("/spaces/:id/children", IsSpaceAdmin(db), api.SpaceChildren)
	router.POST("/spaces/:id/subscriptionPlan", IsSpaceAdmin(db), api.CreateSubscriptionPlan)
	router.GET("/spaces/:id/subscriptionPlan", IsSpaceAdmin(db), api.ListSubscriptionPlans)
	router.GET("/roles", IsTenant, api.ListMyRoles)
	router.Run(":8080")
}

type Api struct {
	db         *gorm.DB
	authClient sdk.Client
}

func (a *Api) SetPassword(ctx *gin.Context) {
	id := ctx.Param("id")
	var req authApi.SetPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	err := a.authClient.SetPassword("tenant", sdk.WithId(id), req.Password)
	if err != nil {
		ctx.JSON(err.StatusCode, err)
		return
	}
	ctx.Status(204)
}

func (a *Api) GetAccount(ctx *gin.Context) {
	id := ctx.Param("id")
	account, err := a.authClient.GetAccount("tenant", sdk.WithId(id))
	if err != nil {
		ctx.JSON(err.StatusCode, err)
		return
	}
	ctx.JSON(200, account)
}

func (a *Api) DeleteTenant(ctx *gin.Context) {
	id := ctx.Param("id")
	err := a.authClient.DeleteAccount("tenant", sdk.WithId(id))
	if err != nil {
		ctx.JSON(err.StatusCode, err)
		return
	}
	ctx.Status(204)
}

func (a *Api) ListTenants(ctx *gin.Context) {
	tenants := a.authClient.ListAccounts("tenant")
	ctx.JSON(200, tenants)
}

func (a *Api) CreateTenant(ctx *gin.Context) {
	tenant := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	if err := ctx.ShouldBindJSON(&tenant); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	newTenant, err := a.authClient.CreateAccount(authApi.CreateAccountRequest{
		Namespace: "tenant",
		Email:     tenant.Email,
		Password:  tenant.Password,
	})
	if err != nil {
		ctx.JSON(err.StatusCode, err)
		return
	}
	ctx.JSON(201, newTenant)
}

func (a *Api) CreateOrganization(ctx *gin.Context) {
	organization := struct {
		Name string `json:"name"`
	}{}
	if err := ctx.ShouldBindJSON(&organization); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	newOrganization := model.Organization{
		Name: organization.Name,
	}
	a.db.Create(&newOrganization)
	ctx.JSON(201, newOrganization)
}

func (a *Api) DeleteOrganization(ctx *gin.Context) {
	id := ctx.Param("id")
	var organization model.Organization
	err := a.db.First(&organization, id).Error
	if err != nil {
		ctx.JSON(404, gin.H{"error": "organization not found"})
		return
	}
	a.db.Delete(&organization)
	ctx.Status(204)
}

func (a *Api) ListMyOrganizations(ctx *gin.Context) {
	ctx.JSON(200, repo.AccountOrganizations(a.db, ctx.MustGet("account").(auth.Account)))
}

func (a *Api) ListAllOrganizations(ctx *gin.Context) {
	var page int64 = 0
	var limit int64 = 10
	if limitString := ctx.Query("limit"); limitString != "" {
		limit, _ = strconv.ParseInt(limitString, 10, 64)
	}
	if pageString := ctx.Query("page"); pageString != "" {
		page, _ = strconv.ParseInt(pageString, 10, 64)
	}
	list, _ := model.ListByOption[model.Organization](a.db, int(limit), int(page))
	ctx.JSON(200, list)
}

func (a *Api) CreateRole(ctx *gin.Context) {
	id := ctx.Param("id")
	role := struct {
		Name string `json:"name"`
	}{}
	if err := ctx.ShouldBindJSON(&role); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	var organization model.Organization
	err := a.db.First(&organization, id).Error
	if err != nil {
		ctx.JSON(404, gin.H{"error": "organization not found"})
		return
	}
	newRole := model.Role{
		OrganizationID: id,
		Name:           role.Name,
	}
	a.db.Create(&newRole)
	ctx.JSON(201, newRole)
}

func (a *Api) ListRole(ctx *gin.Context) {
	id := ctx.Param("id")
	var organization model.Organization
	err := a.db.First(&organization, id).Error
	if err != nil {
		ctx.JSON(404, gin.H{"error": "organization not found"})
		return
	}
	var roles []model.Role
	a.db.Where("organization_id = ?", id).Find(&roles)
	ctx.JSON(200, roles)
}

func (a *Api) DeleteRole(ctx *gin.Context) {
	id := ctx.Param("id")
	err := a.db.Delete(&model.Role{}, id).Error
	if err != nil {
		ctx.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	ctx.Status(204)
}

func (a *Api) SetAccountRole(ctx *gin.Context) {
	id := ctx.Param("id")
	var roleModel model.Role
	err := a.db.First(&roleModel, id).Error
	if err != nil {
		ctx.JSON(404, gin.H{"error": "role not found"})
		return
	}
	accountInfo := struct {
		AccountId string `json:"accountId"`
	}{}
	if err := ctx.ShouldBindJSON(&accountInfo); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	account, e := a.authClient.GetAccount("tenant", sdk.WithId(accountInfo.AccountId))
	if e != nil {
		ctx.JSON(e.StatusCode, e)
		return
	}

	accountRole := model.AccountRole{
		AccountID: account.ID,
		RoleID:    roleModel.ID,
	}
	a.db.Create(&accountRole)
	ctx.Status(204)
}

func (a *Api) ListMyRoles(ctx *gin.Context) {
	account := ctx.MustGet("account").(auth.Account)
	ctx.JSON(200, repo.AccountRoles(a.db, account))
}

func (a *Api) CreateSpace(ctx *gin.Context) {
	var request CreateSpaceRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	organizationId := ctx.Param("id")
	space := model.Space{
		OrganizationID: organizationId,
		Name:           request.Name,
		ParentId:       request.ParentId,
	}
	a.db.Create(&space)
	ctx.JSON(201, space)
}

func (a *Api) ListSpaces(ctx *gin.Context) {
	organizationId := ctx.Param("id")
	var spaces []model.Space
	a.db.Where("organization_id = ?", organizationId).Find(&spaces)
	ctx.JSON(200, spaces)
}

func (a *Api) DeleteSpace(ctx *gin.Context) {
	id := ctx.Param("id")
	var space model.Space
	err := a.db.First(&space, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(404, gin.H{"error": "space not found"})
			return
		}
		ctx.JSON(500, gin.H{"error": "internal server error"})
		return
	}
	a.db.Delete(&space)
	ctx.Status(204)
}

func (a *Api) SpaceChildren(ctx *gin.Context) {
	space := ctx.MustGet("space").(model.Space)
	ctx.JSON(200, space.Children(a.db))
}

func (a *Api) CreateConsumer(ctx *gin.Context) {
	organization := ctx.MustGet("organization").(model.Organization)
	var request CreateConsumerRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	account, err := a.authClient.CreateAccount(authApi.CreateAccountRequest{
		Namespace:   "org/" + organization.ID,
		PhoneRegion: request.PhoneRegion,
		PhoneNumber: request.PhoneNumber,
		Password:    request.Password,
	})
	if err != nil {
		ctx.JSON(err.StatusCode, err)
		return
	}
	ctx.JSON(201, account)
}

func (a *Api) ListConsumer(ctx *gin.Context) {
	organization := ctx.MustGet("organization").(model.Organization)
	ctx.JSON(200, a.authClient.ListAccounts("org/"+organization.ID))
}

func (a *Api) CreateSubscriptionPlan(ctx *gin.Context) {
	space := ctx.MustGet("space").(model.Space)
	var request CreateSubscriptionPlanRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "invalid request"})
		return
	}
	subscriptionPlan, err := space.CreateSubscriptionPlan(a.db, request.Currency, request.Price)
	if err != nil {
		ctx.JSON(500, gin.H{
			"error": "internal server error",
		})
		return
	}
	ctx.JSON(201, subscriptionPlan)
}

func (a *Api) ListSubscriptionPlans(ctx *gin.Context) {
	space := ctx.MustGet("space").(model.Space)
	ctx.JSON(200, space.SubscriptionPlans(a.db))
}
