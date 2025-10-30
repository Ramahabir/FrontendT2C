# QR Code Session Flow - Implementation Summary

## Overview
The Trash2Cash system now uses a secure, session-based QR code mechanism for temporary pairing between recycling stations and users.

## Architecture Changes

### Previous Flow (Login-based)
- QR code was for user authentication
- Station displayed a login QR code
- User scanned to authenticate directly on the station

### New Flow (Session-based)
- QR code represents a temporary session token
- Station requests a session from backend
- User scans to link their authenticated account to the session
- Session expires after 5 minutes to prevent misuse

## Implementation Details

### Backend API Endpoints (Expected)

#### 1. Request Session Token
```
POST /api/request-session

Response:
{
  "success": true,
  "message": "Session token generated",
  "data": {
    "sessionToken": "abc123xyz",
    "qrCode": "data:image/png;base64,...",
    "expiresAt": "2025-10-31T12:05:00Z",
    "status": "pending"
  }
}
```

#### 2. Check Session Status (Station Polling)
```
POST /api/check-session
Body: { "sessionToken": "abc123xyz" }

Response (when user connects):
{
  "success": true,
  "data": {
    "status": "connected",
    "authToken": "user_jwt_token",
    "userId": 123,
    "userName": "John Doe",
    "userBalance": 50000
  }
}
```

#### 3. Connect Session (Mobile App)
```
POST /api/connect-session
Body: {
  "sessionToken": "abc123xyz",
  "authToken": "user_jwt_token"
}

Response:
{
  "success": true,
  "message": "User connected to station session"
}
```

#### 4. End Session
```
POST /api/end-session
Body: { "sessionToken": "abc123xyz" }

Response:
{
  "success": true,
  "message": "Session ended"
}
```

#### 5. Deposit (Updated)
```
POST /api/deposit
Headers: { "Authorization": "Bearer user_jwt_token" }
Body: {
  "material": "plastic",
  "weight": 1.5,
  "sessionToken": "abc123xyz"
}
```

## Frontend Changes

### Go Backend (app.go)

#### App Struct
```go
type App struct {
    ctx            context.Context
    authToken      string        // User's auth token after connection
    currentUserID  int           // Connected user ID
    sessionToken   string        // Current session token
    sessionQRCode  string        // Base64 QR code image
    sessionExpires time.Time     // Session expiration time
    sessionStatus  string        // "pending", "connected", "active", "expired"
}
```

#### New Methods
1. **RequestSessionToken()** - Station requests new session
2. **CheckSessionStatus()** - Station polls for user connection
3. **VerifyAndConnectSession()** - Mobile app connects to session
4. **EndSession()** - Ends current session
5. **clearSession()** - Internal cleanup

#### Updated Methods
- **SubmitTrash()** - Now requires active session
- **StartSensorScan()** - Validates session
- **GetSensorReading()** - Validates session
- **ConfirmSensorSubmission()** - Includes sessionToken

### React Frontend

#### Login.jsx
- Changed from `GenerateQRLoginCode` to `RequestSessionToken`
- Changed from `CheckQRLoginStatus` to `CheckSessionStatus`
- Added session status indicators (pending, connected, expired)
- Updated UI text to reflect station session concept
- Added visual status indicators with animated dots

#### Dashboard.jsx
- Added `EndSession` import and handler
- Changed logout to end session flow
- Added session badge indicator
- Added "End Session" button
- Updated error messages to mention session
- Improved user experience with session awareness

#### CSS Updates (Auth.css)
- Added `.session-status` styles
- Added `.status-indicator` variants (pending, connected, active, expired)
- Added `.status-dot` with pulse animation
- Enhanced visual feedback for session states

#### CSS Updates (Dashboard.css)
- Added `.session-badge` for active session indicator
- Added `.header-actions` for button grouping
- Added `.btn-end-session` styling
- Updated header layout for session UI

## Session States

1. **pending** - Session created, waiting for user to scan
2. **connected** - User has scanned and linked to session
3. **active** - Recycling operations in progress
4. **expired** - Session timeout (5 minutes)

## Security Features

1. **Short-lived tokens** - 5-minute expiration
2. **One-time use** - Session invalidated after use
3. **User authentication** - Mobile app must provide valid auth token
4. **Session validation** - All operations validate active session
5. **Replay attack prevention** - Token expires and cannot be reused

## Mobile App Integration

The mobile app will need to:
1. Implement QR code scanner
2. Parse session token from QR code
3. Send authenticated request to `/station/connect-session`
4. Display connection confirmation to user

Example QR code data format:
```json
{
  "sessionToken": "abc123xyz",
  "expiresAt": "2025-10-31T12:05:00Z"
}
```

## Benefits

1. **Security** - Temporary pairing prevents unauthorized access
2. **Privacy** - No persistent login on station
3. **User Control** - User explicitly connects each session
4. **Session Isolation** - Each recycling session is independent
5. **Audit Trail** - Clear session-based transaction tracking

## Testing Checklist

- [ ] Station can request session token
- [ ] QR code displays correctly
- [ ] Session status polling works
- [ ] Session expires after 5 minutes
- [ ] User can connect via mobile app
- [ ] Connected session allows deposits
- [ ] End session clears all data
- [ ] Multiple sessions don't interfere
- [ ] Error handling for expired sessions
- [ ] UI updates reflect session states

## Notes for Backend Implementation

When implementing the backend:
1. Store sessions in database or cache (Redis recommended)
2. Include station ID in session for multi-station support
3. Clean up expired sessions automatically
4. Implement rate limiting on session creation
5. Log all session events for audit trail
6. Consider WebSocket for real-time updates instead of polling
