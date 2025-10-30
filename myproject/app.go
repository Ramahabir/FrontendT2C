package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://devel-ai.ub.ac.id/api/trash2cash"
)

// App struct
type App struct {
	ctx           context.Context
	authToken     string
	currentUserID int
	qrToken       string
	qrCode        string
	qrExpiresAt   time.Time
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

// APIResponse represents the API response structure
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Helper function to make API requests
func (a *App) makeRequest(method, endpoint string, body interface{}, useAuth bool) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, BaseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if useAuth && a.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.authToken)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp, nil
}

// GenerateQRLoginCode generates a new QR code for login
func (a *App) GenerateQRLoginCode() Response {
	// Check if we have a valid cached QR code
	if a.qrToken != "" && time.Now().Before(a.qrExpiresAt) {
		return Response{
			Success: true,
			Message: "QR code generated",
			Data: map[string]interface{}{
				"token":     a.qrToken,
				"qrCode":    a.qrCode,
				"expiresAt": a.qrExpiresAt.Format(time.RFC3339),
			},
		}
	}

	// Generate new QR code from API
	apiResp, err := a.makeRequest("POST", "/auth/qr-login", nil, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	// Parse the data
	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	// Cache the QR code data
	if token, ok := data["token"].(string); ok {
		a.qrToken = token
	}
	if qrCode, ok := data["qrCode"].(string); ok {
		a.qrCode = qrCode
	}
	if expiresAtStr, ok := data["expiresAt"].(string); ok {
		if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			a.qrExpiresAt = expiresAt
		}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// CheckQRLoginStatus checks if the QR code has been scanned and authenticated
func (a *App) CheckQRLoginStatus(token string) Response {
	reqBody := map[string]string{"token": token}
	apiResp, err := a.makeRequest("POST", "/auth/verify-token", reqBody, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	// Parse the data
	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	// Check if authenticated
	status, _ := data["status"].(string)
	if status == "authenticated" {
		// Store the auth token
		if token, ok := data["token"].(string); ok {
			a.authToken = token
		}
		if userID, ok := data["id"].(float64); ok {
			a.currentUserID = int(userID)
		}

		// Clear cached QR code after successful authentication
		a.qrToken = ""
		a.qrCode = ""
		a.qrExpiresAt = time.Time{}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// AuthenticateQRLogin - This function is typically called by mobile app
// For station app, use CheckQRLoginStatus to poll for authentication
func (a *App) AuthenticateQRLogin(token string, email string, password string) Response {
	return Response{
		Success: false,
		Message: "This function should be called from mobile app, not station app",
	}
}

// Register creates a new user account
func (a *App) Register(req RegisterRequest) Response {
	reqBody := map[string]string{
		"full_name": req.Name,
		"email":     req.Email,
		"password":  req.Password,
	}

	apiResp, err := a.makeRequest("POST", "/auth/register", reqBody, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// Login authenticates a user
func (a *App) Login(req LoginRequest) Response {
	reqBody := map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}

	apiResp, err := a.makeRequest("POST", "/auth/login", reqBody, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	// Store auth token
	if token, ok := data["token"].(string); ok {
		a.authToken = token
	}
	if userID, ok := data["id"].(float64); ok {
		a.currentUserID = int(userID)
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// GetCurrentUser returns the current logged in user's data
func (a *App) GetCurrentUser() Response {
	if a.authToken == "" {
		return Response{Success: false, Message: "Not logged in"}
	}

	apiResp, err := a.makeRequest("GET", "/user/profile", nil, true)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	return Response{
		Success: true,
		Data:    data,
	}
}

// SubmitTrash processes a trash submission (now uses station/deposit endpoint)
func (a *App) SubmitTrash(req SubmissionRequest) Response {
	if a.authToken == "" {
		return Response{Success: false, Message: "Not logged in"}
	}

	reqBody := map[string]interface{}{
		"material": req.Material,
		"weight":   req.Weight,
	}

	apiResp, err := a.makeRequest("POST", "/station/deposit", reqBody, true)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// GetSubmissions returns the submission history for the current user
func (a *App) GetSubmissions() Response {
	if a.authToken == "" {
		return Response{Success: false, Message: "Not logged in"}
	}

	apiResp, err := a.makeRequest("GET", "/transactions", nil, true)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	return Response{
		Success: true,
		Data:    data,
	}
}

// Logout logs out the current user
func (a *App) Logout() Response {
	if a.authToken != "" {
		// Call logout endpoint
		a.makeRequest("POST", "/auth/logout", nil, true)
	}

	a.currentUserID = 0
	a.authToken = ""

	return Response{Success: true, Message: "Logged out successfully"}
}

// StartSensorScan simulates sensor scanning for material and weight
func (a *App) StartSensorScan() Response {
	if a.authToken == "" {
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
	if a.authToken == "" {
		return Response{Success: false, Message: "Not logged in"}
	}

	// Simulate sensor reading with randomized data
	materials := []string{"plastic", "glass", "metal", "paper"}
	materialIndex := time.Now().Unix() % 4
	material := materials[materialIndex]

	// Simulate weight detection (random weight between 0.1 and 5.0 kg)
	weight := 0.1 + float64(time.Now().UnixNano()%50)/10.0

	// Calculate potential reward based on API material rates
	var points float64
	switch material {
	case "plastic":
		points = weight * 10
	case "glass":
		points = weight * 8
	case "metal":
		points = weight * 15
	case "paper":
		points = weight * 5
	}

	// Convert points to rupiah (100 points = Rp 1,000)
	reward := points * 10

	return Response{
		Success: true,
		Message: "Sensor reading complete",
		Data: map[string]interface{}{
			"material": material,
			"weight":   weight,
			"points":   points,
			"reward":   reward,
			"status":   "detected",
		},
	}
}

// ConfirmSensorSubmission processes the sensor-detected trash submission
func (a *App) ConfirmSensorSubmission(material string, weight float64) Response {
	if a.authToken == "" {
		return Response{Success: false, Message: "Not logged in"}
	}

	// Validate input
	if material == "" || weight <= 0 {
		return Response{Success: false, Message: "Invalid sensor data"}
	}

	reqBody := map[string]interface{}{
		"material": material,
		"weight":   weight,
	}

	apiResp, err := a.makeRequest("POST", "/station/deposit", reqBody, true)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// Greet returns a greeting for the given name (keeping for compatibility)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
