package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"myproject/database"
	"time"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// App struct
type App struct {
	ctx           context.Context
	currentUserID int
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	
	// Initialize database
	err := database.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents login data
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SubmissionRequest represents a trash submission
type SubmissionRequest struct {
	Material string  `json:"material"`
	Weight   float64 `json:"weight"`
}

// SensorData represents data from the sensor
type SensorData struct {
	Material string  `json:"material"`
	Weight   float64 `json:"weight"`
	Status   string  `json:"status"` // "detecting", "detected", "ready"
}

// Response is a generic response structure
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// GenerateQRLoginCode generates a new QR code for login
func (a *App) GenerateQRLoginCode() Response {
	// Generate unique session token
	token := uuid.New().String()

	// Set expiration time (5 minutes from now)
	expiresAt := time.Now().Add(5 * time.Minute)

	// Generate QR code image first before inserting into database
	qrBytes, err := qrcode.Encode(token, qrcode.Medium, 256)
	if err != nil {
		return Response{Success: false, Message: "Failed to generate QR code image"}
	}

	// Insert session into database only after QR code is successfully generated
	_, err = database.DB.Exec(
		"INSERT INTO login_sessions (token, status, expires_at) VALUES (?, ?, ?)",
		token, "pending", expiresAt,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to create login session"}
	}

	// Convert to base64 for frontend display
	qrBase64 := base64.StdEncoding.EncodeToString(qrBytes)

	return Response{
		Success: true,
		Message: "QR code generated",
		Data: map[string]interface{}{
			"token":     token,
			"qrCode":    "data:image/png;base64," + qrBase64,
			"expiresAt": expiresAt.Format(time.RFC3339),
		},
	}
}

// CheckQRLoginStatus checks if the QR code has been scanned and authenticated
func (a *App) CheckQRLoginStatus(token string) Response {
	var status string
	var userID sql.NullInt64
	var expiresAt time.Time

	err := database.DB.QueryRow(
		"SELECT status, user_id, expires_at FROM login_sessions WHERE token = ?",
		token,
	).Scan(&status, &userID, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return Response{Success: false, Message: "Invalid session token"}
		}
		return Response{Success: false, Message: "Failed to check status"}
	}

	// Check if token has expired
	if time.Now().After(expiresAt) {
		database.DB.Exec("UPDATE login_sessions SET status = ? WHERE token = ?", "expired", token)
		return Response{
			Success: false,
			Message: "QR code has expired",
			Data: map[string]interface{}{
				"status": "expired",
			},
		}
	}

	// If authenticated, get user data and log them in
	if status == "authenticated" && userID.Valid {
		var user database.User
		err := database.DB.QueryRow(
			"SELECT id, full_name, email, total_points FROM users WHERE id = ?",
			userID.Int64,
		).Scan(&user.ID, &user.FullName, &user.Email, &user.TotalPoints)

		if err != nil {
			return Response{Success: false, Message: "Failed to retrieve user data"}
		}

		// Set current user
		a.currentUserID = user.ID

		// Clean up session
		database.DB.Exec("DELETE FROM login_sessions WHERE token = ?", token)

		return Response{
			Success: true,
			Message: "Login successful",
			Data: map[string]interface{}{
				"status":  "authenticated",
				"id":      user.ID,
				"name":    user.FullName,
				"email":   user.Email,
				"balance": user.TotalPoints,
			},
		}
	}

	return Response{
		Success: true,
		Message: "Waiting for authentication",
		Data: map[string]interface{}{
			"status": status,
		},
	}
}

// AuthenticateQRLogin simulates mobile app authenticating the QR code
// In production, this would be called by the mobile app backend
func (a *App) AuthenticateQRLogin(token string, email string, password string) Response {
	// Verify the session exists and is pending
	var sessionID int
	var status string
	var expiresAt time.Time

	err := database.DB.QueryRow(
		"SELECT id, status, expires_at FROM login_sessions WHERE token = ?",
		token,
	).Scan(&sessionID, &status, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return Response{Success: false, Message: "Invalid QR code"}
		}
		return Response{Success: false, Message: "Failed to verify QR code"}
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return Response{Success: false, Message: "QR code has expired"}
	}

	// Check if already used
	if status != "pending" {
		return Response{Success: false, Message: "QR code has already been used"}
	}

	// Authenticate user credentials
	var user database.User
	var hashedPassword string

	err = database.DB.QueryRow(
		"SELECT id, full_name, email, password, total_points FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.FullName, &user.Email, &hashedPassword, &user.TotalPoints)

	if err != nil {
		if err == sql.ErrNoRows {
			return Response{Success: false, Message: "Invalid email or password"}
		}
		return Response{Success: false, Message: "Authentication failed"}
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return Response{Success: false, Message: "Invalid email or password"}
	}

	// Update session with user ID and mark as authenticated
	_, err = database.DB.Exec(
		"UPDATE login_sessions SET user_id = ?, status = ? WHERE id = ?",
		user.ID, "authenticated", sessionID,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to complete authentication"}
	}

	return Response{
		Success: true,
		Message: "QR code authenticated successfully",
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"id":    user.ID,
				"name":  user.FullName,
				"email": user.Email,
			},
		},
	}
}

// Register creates a new user account
func (a *App) Register(req RegisterRequest) Response {
	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return Response{Success: false, Message: "All fields are required"}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Response{Success: false, Message: "Failed to process password"}
	}

	// Insert user into database
	result, err := database.DB.Exec(
		"INSERT INTO users (full_name, email, password, total_points) VALUES (?, ?, ?, ?)",
		req.Name, req.Email, string(hashedPassword), 0,
	)
	if err != nil {
		return Response{Success: false, Message: "Email already exists"}
	}

	userID, _ := result.LastInsertId()
	
	return Response{
		Success: true,
		Message: "Registration successful",
		Data: map[string]interface{}{
			"id":    userID,
			"name":  req.Name,
			"email": req.Email,
		},
	}
}

// Login authenticates a user
func (a *App) Login(req LoginRequest) Response {
	var user database.User
	var hashedPassword string

	err := database.DB.QueryRow(
		"SELECT id, full_name, email, password, total_points FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.FullName, &user.Email, &hashedPassword, &user.TotalPoints)

	if err != nil {
		if err == sql.ErrNoRows {
			return Response{Success: false, Message: "Invalid email or password"}
		}
		return Response{Success: false, Message: "Login failed"}
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password))
	if err != nil {
		return Response{Success: false, Message: "Invalid email or password"}
	}

	// Set current user
	a.currentUserID = user.ID

	return Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"id":      user.ID,
			"name":    user.FullName,
			"email":   user.Email,
			"balance": user.TotalPoints,
		},
	}
}

// GetCurrentUser returns the current logged in user's data
func (a *App) GetCurrentUser() Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	var user database.User
	err := database.DB.QueryRow(
		"SELECT id, full_name, email, total_points FROM users WHERE id = ?",
		a.currentUserID,
	).Scan(&user.ID, &user.FullName, &user.Email, &user.TotalPoints)

	if err != nil {
		return Response{Success: false, Message: "User not found"}
	}

	return Response{
		Success: true,
		Data: map[string]interface{}{
			"id":      user.ID,
			"name":    user.FullName,
			"email":   user.Email,
			"balance": user.TotalPoints,
		},
	}
}

// SubmitTrash processes a trash submission and calculates reward
func (a *App) SubmitTrash(req SubmissionRequest) Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	// Validate input
	if req.Material == "" || req.Weight <= 0 {
		return Response{Success: false, Message: "Invalid material or weight"}
	}

	// Calculate reward based on material type
	var reward float64
	switch req.Material {
	case "plastic":
		reward = req.Weight * 5000 // Rp 5,000 per kg
	case "metal":
		reward = req.Weight * 10000 // Rp 10,000 per kg
	case "paper":
		reward = req.Weight * 2000 // Rp 2,000 per kg
	default:
		return Response{Success: false, Message: "Invalid material type"}
	}

	// Begin transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return Response{Success: false, Message: "Failed to process submission"}
	}
	defer tx.Rollback()

	// Insert submission
	_, err = tx.Exec(
		"INSERT INTO submissions (user_id, material, weight, reward) VALUES (?, ?, ?, ?)",
		a.currentUserID, req.Material, req.Weight, reward,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to save submission"}
	}

	// Update user balance
	_, err = tx.Exec(
		"UPDATE users SET balance = balance + ? WHERE id = ?",
		reward, a.currentUserID,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to update balance"}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return Response{Success: false, Message: "Failed to complete submission"}
	}

	// Get updated balance
	var newBalance float64
	database.DB.QueryRow("SELECT balance FROM users WHERE id = ?", a.currentUserID).Scan(&newBalance)

	return Response{
		Success: true,
		Message: fmt.Sprintf("Submission successful! You earned Rp %.0f", reward),
		Data: map[string]interface{}{
			"reward":     reward,
			"newBalance": newBalance,
		},
	}
}

// GetSubmissions returns the submission history for the current user
func (a *App) GetSubmissions() Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	rows, err := database.DB.Query(
		"SELECT id, material, weight, reward, created_at FROM submissions WHERE user_id = ? ORDER BY created_at DESC",
		a.currentUserID,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to retrieve submissions"}
	}
	defer rows.Close()

	var submissions []map[string]interface{}
	for rows.Next() {
		var id int
		var material string
		var weight, reward float64
		var createdAt time.Time

		err := rows.Scan(&id, &material, &weight, &reward, &createdAt)
		if err != nil {
			continue
		}

		submissions = append(submissions, map[string]interface{}{
			"id":        id,
			"material":  material,
			"weight":    weight,
			"reward":    reward,
			"createdAt": createdAt.Format("2006-01-02 15:04:05"),
		})
	}

	if submissions == nil {
		submissions = []map[string]interface{}{}
	}

	return Response{
		Success: true,
		Data:    submissions,
	}
}

// Logout logs out the current user
func (a *App) Logout() Response {
	a.currentUserID = 0
	return Response{Success: true, Message: "Logged out successfully"}
}

// StartSensorScan simulates sensor scanning for material and weight
func (a *App) StartSensorScan() Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	return Response{
		Success: true,
		Message: "Sensor scan started",
		Data: map[string]interface{}{
			"status": "detecting",
		},
	}
}

// GetSensorReading simulates getting real-time sensor data
func (a *App) GetSensorReading() Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	// Simulate sensor reading with randomized data
	materials := []string{"plastic", "metal", "paper"}
	materialIndex := time.Now().Unix() % 3
	material := materials[materialIndex]
	
	// Simulate weight detection (random weight between 0.1 and 5.0 kg)
	weight := 0.1 + float64(time.Now().UnixNano()%50)/10.0
	
	// Calculate potential reward
	var reward float64
	switch material {
	case "plastic":
		reward = weight * 5000
	case "metal":
		reward = weight * 10000
	case "paper":
		reward = weight * 2000
	}

	return Response{
		Success: true,
		Message: "Sensor reading complete",
		Data: map[string]interface{}{
			"material": material,
			"weight":   weight,
			"reward":   reward,
			"status":   "detected",
		},
	}
}

// ConfirmSensorSubmission processes the sensor-detected trash submission
func (a *App) ConfirmSensorSubmission(material string, weight float64) Response {
	if a.currentUserID == 0 {
		return Response{Success: false, Message: "Not logged in"}
	}

	// Validate input
	if material == "" || weight <= 0 {
		return Response{Success: false, Message: "Invalid sensor data"}
	}

	// Calculate reward based on material type
	var reward float64
	switch material {
	case "plastic":
		reward = weight * 5000 // Rp 5,000 per kg
	case "metal":
		reward = weight * 10000 // Rp 10,000 per kg
	case "paper":
		reward = weight * 2000 // Rp 2,000 per kg
	default:
		return Response{Success: false, Message: "Invalid material type"}
	}

	// Begin transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return Response{Success: false, Message: "Failed to process submission"}
	}
	defer tx.Rollback()

	// Insert submission
	_, err = tx.Exec(
		"INSERT INTO submissions (user_id, material, weight, reward) VALUES (?, ?, ?, ?)",
		a.currentUserID, material, weight, reward,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to save submission"}
	}

	// Update user balance
	_, err = tx.Exec(
		"UPDATE users SET balance = balance + ? WHERE id = ?",
		reward, a.currentUserID,
	)
	if err != nil {
		return Response{Success: false, Message: "Failed to update balance"}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return Response{Success: false, Message: "Failed to complete submission"}
	}

	// Get updated balance
	var newBalance float64
	database.DB.QueryRow("SELECT balance FROM users WHERE id = ?", a.currentUserID).Scan(&newBalance)

	return Response{
		Success: true,
		Message: fmt.Sprintf("Submission successful! You earned Rp %.0f", reward),
		Data: map[string]interface{}{
			"material":   material,
			"weight":     weight,
			"reward":     reward,
			"newBalance": newBalance,
		},
	}
}

// Greet returns a greeting for the given name (keeping for compatibility)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
