package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type DBClient struct {
	Conn *sql.DB
}

func InitMySQL() (*DBClient, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Println("✅ Connected to MySQL Audit Database")
	return &DBClient{Conn: db}, nil
}

// LogEvent writes the audit log to the database
func (c *DBClient) LogEvent(action, uid, group string, expiry int64, status, details string) {
	query := `INSERT INTO access_audit_logs (action_type, target_uid, target_group, expiry_time, status, details) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	var expiryTime string
	if expiry > 0 {
		expiryTime = time.Unix(expiry, 0).Format("2006-01-02 15:04:05")
	}
	_, err := c.Conn.Exec(query, action, uid, group, expiryTime, status, details)
	if err != nil {
		log.Printf("❌ CRITICAL: Failed to write audit log to MySQL: %v", err)
	}
}
