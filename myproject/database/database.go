package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never send password to frontend
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

// Submission represents a trash submission
type Submission struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Material   string    `json:"material"`
	Weight     float64   `json:"weight"`
	Reward     float64   `json:"reward"`
	CreatedAt  time.Time `json:"created_at"`
}

// InitDB initializes the database connection and creates tables
func InitDB() error {
	var err error
	DB, err = sql.Open("sqlite3", "./trash2cash.db")
	if err != nil {
		return err
	}

	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		balance REAL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = DB.Exec(createUsersTable)
	if err != nil {
		return err
	}

	// Create submissions table
	createSubmissionsTable := `
	CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		material TEXT NOT NULL,
		weight REAL NOT NULL,
		reward REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	_, err = DB.Exec(createSubmissionsTable)
	if err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
