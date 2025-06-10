package dao

import (
	"fmt"
	"taskd/internal/utils"

	"gorm.io/driver/mysql"
	// "gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DbTypeMySQL  = "mysql"
	DbTypeSQLite = "sqlite3"
)

var DB *gorm.DB

/*
 * initMySQL initializes a MySQL database connection
 */
func initMySQL(dbc utils.DbConfig) error {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local",
		dbc.User, dbc.Password, dbc.Host, dbc.Port, dbc.DatabaseName)
	var err error
	if DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}); err != nil {
		return err
	}
	return nil
}

/*
 * initSqlite initializes a SQLite database connection
 */
func initSqlite(dbc utils.DbConfig) error {
	name := dbc.DatabaseName
	if name == "" {
		name = "taskd"
	}
	name = fmt.Sprintf("%s.db", name)
	var err error
	DB, err = gorm.Open(sqlite.Open(name), &gorm.Config{})
	if err != nil {
		return err
	}
	return err
}

/*
 * InitDB initializes database connection based on configuration
 * Supports MySQL and SQLite database types
 */
func InitDB(dbc utils.DbConfig) error {
	if dbc.Type == DbTypeMySQL {
		if err := initMySQL(dbc); err != nil {
			return fmt.Errorf("failed to initialize MySQL database: %v", err)
		}
	} else if dbc.Type == DbTypeSQLite {
		if err := initSqlite(dbc); err != nil {
			return fmt.Errorf("failed to initialize SQLite database: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported database type: %s", dbc.Type)
	}

	DB.AutoMigrate(&TemplateRec{})
	DB.AutoMigrate(&Pool{})
	DB.AutoMigrate(&PoolResource{})
	return nil
}
