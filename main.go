package main

import (
	"fmt"
	"os"

	"github.com/alterminal/auth/sdk"
	"github.com/alterminal/member/api"
	"github.com/alterminal/member/repo"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-go/v81"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func init() {
	path := os.Getenv("config")
	if path == "" {
		path = "."
	}
	fmt.Println(path)
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("yaml")
	viper.ReadInConfig()
	stripe.Key = viper.GetString("stripe.key")
}

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.host"),
		viper.GetString("database.port"),
		viper.GetString("database.database"))
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	authClient := sdk.Client{
		BaseUrl:     viper.GetString("auth.baseUrl"),
		AccessToken: viper.GetString("auth.accessToken"),
	}
	repo.Init(db)
	api.Run(db, authClient)
}
