package api

import (
	"strings"

	authApi "github.com/alterminal/auth/api"
	auth "github.com/alterminal/auth/model"
	"github.com/alterminal/auth/sdk"
	"github.com/alterminal/common/mid"
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
	router.POST("/organizations", IsAdmin, api.CreateOrganization)
	router.Run(":8080")
}

type Api struct {
	db         *gorm.DB
	authClient sdk.Client
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
	ctx.JSON(204, nil)
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
}
