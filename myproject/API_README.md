# Trash 2 Cash - API Documentation

## Overview
Trash 2 Cash is a comprehensive recycling management system with both a Wails desktop app and REST API backend.

## Architecture
- **Backend**: Go REST API server (Port 8080)
- **Frontend**: Wails Windows Desktop Application
- **Database**: SQLite3
- **Session Model**: Token-based temporary pairing between station and user

## QR Code Session Flow

The Trash2Cash system uses a secure session-based QR code mechanism:

1. **Station Request**: Station requests a session token from backend (`/api/request-session`)
2. **QR Display**: Backend generates a short-lived session token and returns it as a QR code
3. **User Scan**: User scans QR code with mobile app
4. **Session Linking**: Mobile app sends token + user auth to backend (`/api/connect-session`)
5. **Verification**: Backend verifies and links user to station session
6. **Active Session**: Station polls status (`/api/check-session`) and proceeds with recycling
7. **Session Expiry**: Token expires after 5 minutes to prevent replay attacks

## API Endpoints

### Station Session Management
```
POST   /api/request-session  - Request new session token and QR code
POST   /api/check-session    - Check if user has connected to session
POST   /api/connect-session  - Connect user to station session (mobile app)
POST   /api/end-session      - End the current recycling session
POST   /api/deposit          - Process item deposit (requires active session)
GET    /api/status           - Get station status and today's stats
GET    /api/config           - Get station configuration
```

### Authentication
```
POST   /api/auth/register       - Register new user
POST   /api/auth/login          - Login with email/password (mobile app)
POST   /api/auth/logout         - Logout and invalidate session
```

### User Management (Protected)
```
GET    /api/user/profile        - Get user profile
GET    /api/user/stats          - Get recycling statistics
PUT    /api/user/profile        - Update user information
```

### Transactions (Protected)
```
GET    /api/transactions        - Get transaction history (with pagination)
POST   /api/transactions        - Create new transaction (manual entry)
GET    /api/transactions/{id}   - Get specific transaction details
```

### Redemptions (Protected)
```
GET    /api/redemption/options  - Get available redemption methods
POST   /api/redemption/redeem   - Redeem points for cash/vouchers
GET    /api/redemption/history  - Get redemption history
```

### Station Management (Protected)
```
GET    /api/station/status      - Get station status and today's stats
POST   /api/station/deposit     - Process item deposit
GET    /api/station/config      - Get station configuration
```

### WebSocket
```
WS     /ws                      - Real-time updates
```

### Health Check
```
GET    /api/health              - API health check
```

## Material Points Calculation

| Material | Points per KG |
|----------|---------------|
| Plastic  | 10 points     |
| Glass    | 8 points      |
| Metal    | 15 points     |
| Paper    | 5 points      |

## Redemption Rates
- **100 points = Rp 1,000**
- Minimum redemption:
  - Bank Transfer: 1000 points
  - Cash Pickup: 500 points
  - Voucher: 250 points

## Authentication
Protected endpoints require JWT token in Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

## Database Schema

### Users
- id, email, password, full_name, phone, total_points, created_at, updated_at

### Transactions
- id, user_id, type, amount, item_type, weight, points_earned, station_id, timestamp

### Redemptions
- id, user_id, points_used, amount_cash, method, status, account_info, timestamp

### Sessions
- id, user_id, token, qr_token, expires_at, created_at

### Stations
- id, location, status, capacity, last_maintenance, configuration

## Running the Application

### Start the Application
```bash
wails dev
```

This will start:
- API Server on `http://localhost:8080`
- Wails Desktop App

### Test the API
```bash
# Health check
curl http://localhost:8080/api/health

# Register user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","full_name":"John Doe"}'

# Generate QR login
curl -X POST http://localhost:8080/api/auth/qr-login
```

## Frontend Features
1. **Station Control Dashboard** - Live monitoring and statistics
2. **QR Code Display** - For mobile authentication
3. **Deposit Processing** - Process recyclable items
4. **Admin Panel** - User and transaction management

## Future Enhancements
- Mobile app integration
- E-wallet integration
- Analytics dashboard
- Multi-station support
- Cloud synchronization
