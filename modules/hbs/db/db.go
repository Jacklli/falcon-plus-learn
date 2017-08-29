package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/open-falcon/falcon-plus/modules/hbs/g"
	"log"
)

/*
import _ "github.com/go-sql-driver/mysql" 为了执行init函数

#  mysql/driver.go
func init() {
	sql.Register("mysql", &MySQLDriver{})
}
 */

var DB *sql.DB
/*
连接数据库
 */
func Init() {
	var err error
	DB, err = sql.Open("mysql", g.Config().Database) // DSN: username:password@protocol(address)/dbname?param=value
	if err != nil {
		log.Fatalln("open db fail:", err)
	}

	DB.SetMaxOpenConns(g.Config().MaxConns)
	DB.SetMaxIdleConns(g.Config().MaxIdle)

	err = DB.Ping()
	if err != nil {
		log.Fatalln("ping db fail:", err)
	}
}
