package mysqlserver

import (
	"database/sql"
	"net"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

// MysqlClient Session
var MysqlClient *sql.DB

// GetMysqlSession start redis
func GetMysqlSession() *sql.DB {
	if MysqlClient == nil {
		logrus.Info("Starting mysql session!")
		Initialize()
		return MysqlClient
	} else {
		return MysqlClient
	}
}

// Initialize a redis instance
func Initialize() {
	connection := os.Getenv("DB_HOST")
	username := os.Getenv("MYSQLUSER")
	password := os.Getenv("MYSQLPASS")

	for {
		conn, err := net.DialTimeout("tcp", connection, 6*time.Second)
		if conn != nil {
			conn.Close()
			break
		}

		logrus.Info("Sleeping till mysql be available... ", err)
		time.Sleep(5 * time.Second)
	}

	db, err := sql.Open("mysql", ""+username+":"+password+"@tcp("+connection+")/zinnion?charset=utf8")
	if err != nil {
		logrus.Info("No connection with mysql, ", err)
	}
	MysqlClient = db
}
