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
	ctx            context.Context
	authToken      string
	currentUserID  int
	sessionToken   string
	sessionQRCode  string
	sessionExpires time.Time
	sessionStatus  string // "pending", "connected", "active", "expired"
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

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check if response is empty
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("empty response from server (status: %d)", resp.StatusCode)
	}

	// Check for non-200 status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v (body: %s)", err, string(bodyBytes))
	}

	return &apiResp, nil
}

// RequestSessionToken - Station requests a new session token from backend
// This initiates a recycling session that users can connect to via QR code
func (a *App) RequestSessionToken() Response {
	// Check if we have a valid active session
	if a.sessionToken != "" && time.Now().Before(a.sessionExpires) && a.sessionStatus != "expired" {
		return Response{
			Success: true,
			Message: "Active session token",
			Data: map[string]interface{}{
				"sessionToken": a.sessionToken,
				"qrCode":       a.sessionQRCode,
				"expiresAt":    a.sessionExpires.Format(time.RFC3339),
				"status":       a.sessionStatus,
			},
		}
	}

	// Request new session token from backend
	apiResp, err := a.makeRequest("POST", "/request-session", nil, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	// Parse the response data
	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	// Cache the session data
	if token, ok := data["sessionToken"].(string); ok {
		a.sessionToken = token
	}
	if qrCode, ok := data["qrCode"].(string); ok {
		a.sessionQRCode = qrCode
	}
	if expiresAtStr, ok := data["expiresAt"].(string); ok {
		if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			a.sessionExpires = expiresAt
		}
	}
	a.sessionStatus = "pending" // waiting for user to scan

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// CheckSessionStatus - Station polls to check if a user has connected to the session
// Returns the session status and user information if connected
func (a *App) CheckSessionStatus() Response {
	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session. Please request a session token first."}
	}

	// Check if session has expired locally
	if time.Now().After(a.sessionExpires) {
		a.sessionStatus = "expired"
		return Response{
			Success: false,
			Message: "Session expired",
			Data: map[string]interface{}{
				"status": "expired",
			},
		}
	}

	// Poll backend for session status
	reqBody := map[string]string{"sessionToken": a.sessionToken}
	apiResp, err := a.makeRequest("POST", "/check-session", reqBody, false)
	if err != nil {
		return Response{Success: false, Message: "Failed to connect to server: " + err.Error()}
	}

	if !apiResp.Success {
		return Response{Success: false, Message: apiResp.Message}
	}

	// Parse the response data
	var data map[string]interface{}
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return Response{Success: false, Message: "Failed to parse response"}
	}

	// Update session status
	if status, ok := data["status"].(string); ok {
		a.sessionStatus = status

		// If user connected, store auth token and user info
		if status == "connected" || status == "active" {
			if token, ok := data["authToken"].(string); ok {
				a.authToken = token
			}
			if userID, ok := data["userId"].(float64); ok {
				a.currentUserID = int(userID)
			}
		}
	}

	return Response{
		Success: true,
		Message: apiResp.Message,
		Data:    data,
	}
}

// VerifyAndConnectSession - Called by mobile app to verify session token and connect user
// This links the authenticated user to the station's active session
func (a *App) VerifyAndConnectSession(sessionToken string, userAuthToken string) Response {
	reqBody := map[string]string{
		"sessionToken": sessionToken,
		"authToken":    userAuthToken,
	}

	apiResp, err := a.makeRequest("POST", "/connect-session", reqBody, false)
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

// EndSession - Ends the current station session and clears session data
func (a *App) EndSession() Response {
	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session"}
	}

	// Notify backend to end session
	reqBody := map[string]string{"sessionToken": a.sessionToken}
	_, err := a.makeRequest("POST", "/end-session", reqBody, false)
	if err != nil {
		// Clear local session even if backend call fails
		a.clearSession()
		return Response{Success: false, Message: "Failed to notify server, but session cleared locally: " + err.Error()}
	}

	// Clear local session data
	a.clearSession()

	return Response{
		Success: true,
		Message: "Session ended successfully",
	}
}

// clearSession - Internal helper to clear session data
func (a *App) clearSession() {
	a.sessionToken = ""
	a.sessionQRCode = ""
	a.sessionExpires = time.Time{}
	a.sessionStatus = ""
	a.authToken = ""
	a.currentUserID = 0
}

// SubmitTrash processes a trash submission (now uses station/deposit endpoint with session)
func (a *App) SubmitTrash(req SubmissionRequest) Response {
	if a.authToken == "" {
		return Response{Success: false, Message: "No user connected. Please scan QR code first."}
	}

	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session. Please request a session token first."}
	}

	reqBody := map[string]interface{}{
		"material":     req.Material,
		"weight":       req.Weight,
		"sessionToken": a.sessionToken,
	}

	apiResp, err := a.makeRequest("POST", "/deposit", reqBody, true)
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
		return Response{Success: false, Message: "No user connected. Please scan QR code first."}
	}

	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session."}
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
		return Response{Success: false, Message: "No user connected. Please scan QR code first."}
	}

	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session."}
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
		return Response{Success: false, Message: "No user connected. Please scan QR code first."}
	}

	if a.sessionToken == "" {
		return Response{Success: false, Message: "No active session."}
	}

	// Validate input
	if material == "" || weight <= 0 {
		return Response{Success: false, Message: "Invalid sensor data"}
	}

	reqBody := map[string]interface{}{
		"material":     material,
		"weight":       weight,
		"sessionToken": a.sessionToken,
	}

	apiResp, err := a.makeRequest("POST", "/deposit", reqBody, true)
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
