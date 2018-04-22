package g

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"  // 导入driver
	"log"
	"sync"
)

// TODO 草草的写了一个db连接池,优化下
var (
	dbLock    sync.RWMutex  // 用于连接池的互斥访问
	dbConnMap map[string]*sql.DB
)

// database/sql包的使用方法请参考：https://segmentfault.com/a/1190000003036452
var DB *sql.DB // sql.DB不是一个连接，它是数据库的抽象接口。它可以根据driver打开关闭数据库连接，管理连接池。

/*
创建数据库连接DB *sql.DB，看是否能成功创建，初始化连接池dbConnMap
 */
func InitDB() {
	var err error
	DB, err = makeDbConn() // 创建一个新的mysql连接
	if DB == nil || err != nil {
		log.Fatalln("g.InitDB, get db conn fail", err)
	}

	dbConnMap = make(map[string]*sql.DB) // 用于存储连接的map
	log.Println("g.InitDB ok")
}

/*
根据connName从连接池dbConnMap查找已建立连接并返回
如果不存在，则创建并加入连接池
 */
func GetDbConn(connName string) (c *sql.DB, e error) {
	dbLock.Lock()
	defer dbLock.Unlock()

	var err error
	var dbConn *sql.DB
	dbConn = dbConnMap[connName]
	if dbConn == nil {  // 不存在，则新建数据库连接，并保存到dbConnMap
		dbConn, err = makeDbConn()
		if dbConn == nil || err != nil {
			closeDbConn(dbConn)
			return nil, err
		}
		dbConnMap[connName] = dbConn
	}

	err = dbConn.Ping() // 存在，验证连接
	if err != nil {
		closeDbConn(dbConn)
		delete(dbConnMap, connName)
		return nil, err
	}

	return dbConn, err
}

// 创建一个新的mysql连接
func makeDbConn() (conn *sql.DB, err error) {
	conn, err = sql.Open("mysql", Config().DB.Dsn)  // Dsn: "用户名:密码@tcp(IP:端口)/数据库?charset=utf8"
	if err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(Config().DB.MaxIdle) // 设置空闲连接数
	err = conn.Ping()

	return conn, err
}

/*
关闭连接
 */
func closeDbConn(conn *sql.DB) {
	if conn != nil {
		conn.Close()
	}
}
