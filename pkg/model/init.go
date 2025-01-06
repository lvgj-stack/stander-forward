package model

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Mr-LvGJ/stander/pkg/config"
	"github.com/Mr-LvGJ/stander/pkg/model/dal"
)

var (
	internalDb *gorm.DB
)

func InitMysql(c *config.Database) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.Username, c.Password, c.Addr, c.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	internalDb = db

	//if err := db.AutoMigrate(&entity.Rule{}, &entity.Chain{}, &entity.Node{}); err != nil {
	//	panic(err)
	//}

	dal.SetDefault(internalDb)
}

func GetDb() *gorm.DB {
	return internalDb
}
