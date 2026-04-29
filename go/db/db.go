package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Conn is the global database connection pool.
var Conn *sql.DB

// Init opens a MariaDB connection pool.
// DSN can be overridden via the MIKOMIK_DSN environment variable.
func Init() {
	dsn := os.Getenv("MIKOMIK_DSN")
	if dsn == "" {
		dsn = "root:@tcp(127.0.0.1:3306)/mikomik?parseTime=true"
	}

	var err error
	Conn, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to open DB: %v", err)
	}

	if err = Conn.Ping(); err != nil {
		log.Printf("⚠️  MariaDB not reachable (%v). Auth features will be unavailable.", err)
		Conn = nil
		return
	}

	Conn.SetMaxOpenConns(25)
	Conn.SetMaxIdleConns(5)

	fmt.Println("✅ Connected to MariaDB")
}
