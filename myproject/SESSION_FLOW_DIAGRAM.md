# Trash2Cash Session Flow Diagram

## Complete Flow Visualization

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     TRASH2CASH SESSION FLOW                              │
└─────────────────────────────────────────────────────────────────────────┘

   STATION APP              BACKEND API              MOBILE APP
   ───────────              ───────────              ──────────
       │                         │                        │
       │                         │                        │
   ┌───┴───┐                     │                        │
   │ START │                     │                        │
   │SESSION│                     │                        │
   └───┬───┘                     │                        │
       │                         │                        │
       │ 1. Request Session      │                        │
       ├────────────────────────>│                        │
       │   POST /api/            │                        │
       │   request-session       │                        │
       │                         │                        │
       │ 2. Session Token + QR   │                        │
       │<────────────────────────┤                        │
       │   {sessionToken,        │                        │
       │    qrCode, expiresAt}   │                        │
       │                         │                        │
   ┌───┴───┐                     │                        │
   │DISPLAY│                     │                        │
   │QR CODE│                     │                        │
   └───┬───┘                     │                        │
       │                         │                        │
       │ 3. Poll Session Status  │                        │
       ├────────────────────────>│                        │
       │   POST /api/            │                        │
       │   check-session         │                        │
       │   (every 2 seconds)     │                        │
       │                         │                        │
       │ Response: pending       │                        │
       │<────────────────────────┤                        │
       │   {status: "pending"}   │                        │
       │                         │                        │
       │                         │    ┌──────────┐        │
       │                         │    │  USER    │        │
       │                         │    │ SCANS QR │        │
       │                         │    └────┬─────┘        │
       │                         │         │              │
       │                         │         v              │
       │                         │    ┌────┴────┐        │
       │                         │    │MOBILE   │        │
       │                         │    │APP      │        │
       │                         │    │OPENS    │        │
       │                         │    └────┬────┘        │
       │                         │         │              │
       │                         │     4. Connect Session │
       │                         │<────────┼──────────────┤
       │                         │         │   POST       │
       │                         │         │   /api/      │
       │                         │         │   connect-   │
       │                         │         │   session    │
       │                         │         │   {token,    │
       │                         │         │    userAuth} │
       │                         │         │              │
       │                         │ 5. Verify & Link       │
       │                         │    User to Session     │
       │                         │         │              │
       │                         │     6. Success         │
       │                         ├────────┼──────────────>│
       │                         │         │   {success}  │
       │                         │         │              │
       │ 7. Poll (connected!)    │         │              │
       ├────────────────────────>│         │              │
       │                         │         │              │
       │ 8. User Connected!      │         │              │
       │<────────────────────────┤         │              │
       │   {status: "connected", │         │              │
       │    authToken, userId,   │         │              │
       │    userName}            │         │              │
       │                         │         │              │
   ┌───┴────┐                    │         │              │
   │SHOW    │                    │         │              │
   │DASH    │                    │         │              │
   │BOARD   │                    │         │              │
   └───┬────┘                    │         │              │
       │                         │         │              │
       │ 9. User Deposits Items  │         │              │
       ├────────────────────────>│         │              │
       │   POST /api/deposit     │         │              │
       │   {material, weight,    │         │              │
       │    sessionToken}        │         │              │
       │   Header: Bearer token  │         │              │
       │                         │         │              │
       │ 10. Points Added        │         │              │
       │<────────────────────────┤         │              │
       │   {newBalance, points}  │         │              │
       │                         │         │              │
       │         ...             │         │              │
       │   (multiple deposits)   │         │              │
       │         ...             │         │              │
       │                         │         │              │
       │ 11. End Session         │         │              │
       ├────────────────────────>│         │              │
       │   POST /station/        │         │              │
       │   end-session           │         │              │
       │                         │         │              │
       │ 12. Session Ended       │         │              │
       │<────────────────────────┤         │              │
       │   {success}             │         │              │
       │                         │         │              │
   ┌───┴────┐                    │         │              │
   │RETURN  │                    │         │              │
   │TO QR   │                    │         │              │
   │SCREEN  │                    │         │              │
   └────────┘                    │         │              │
```

## Session Lifecycle

```
┌─────────────┐
│   CREATED   │ ← Station requests session
└──────┬──────┘
       │
       ↓ (QR displayed)
┌─────────────┐
│   PENDING   │ ← Waiting for user scan
└──────┬──────┘
       │
       ↓ (User scans & connects)
┌─────────────┐
│  CONNECTED  │ ← User linked to session
└──────┬──────┘
       │
       ↓ (Deposits being made)
┌─────────────┐
│   ACTIVE    │ ← Recycling in progress
└──────┬──────┘
       │
       ↓ (End session or timeout)
┌─────────────┐
│   EXPIRED   │ ← Session ended
└─────────────┘
```

## Security Timeline

```
Time:    0:00          1:00          2:00          3:00          4:00          5:00
         │             │             │             │             │             │
         │◄────────────────────── Session Valid ──────────────────────────────►│
         │                                                                     │
    ┌────┴────┐                                                          ┌────┴────┐
    │ SESSION │                                                          │ SESSION │
    │ CREATED │                                                          │ EXPIRES │
    └─────────┘                                                          └─────────┘
         │                                                                     │
         │                                                                     │
         ↓ QR Displayed                                                       ↓ Auto-cleanup
    ┌─────────────────────────────────────────────────────────────────────────┐
    │  User must scan and complete recycling within 5 minutes                 │
    └─────────────────────────────────────────────────────────────────────────┘
```

## Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                        SESSION DATA                               │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  sessionToken:   "xyz789"                                        │
│  stationId:      "STATION-001"                                   │
│  status:         "pending" → "connected" → "active" → "expired"  │
│  userId:         null → 123 (after connection)                   │
│  authToken:      null → "jwt_token" (after connection)           │
│  createdAt:      "2025-10-31T12:00:00Z"                         │
│  expiresAt:      "2025-10-31T12:05:00Z"                         │
│  lastActivity:   "2025-10-31T12:02:30Z"                         │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

## Error Handling

```
┌─────────────────────────────────────────────────────────────┐
│                    ERROR SCENARIOS                           │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Session Expired                                         │
│     → Auto-generate new QR code                            │
│     → Show "Session expired" message                        │
│                                                              │
│  2. Invalid Session Token                                   │
│     → Return error to mobile app                            │
│     → Request new QR scan                                   │
│                                                              │
│  3. User Already Connected                                  │
│     → Prevent duplicate connections                         │
│     → Show "Session already active" error                   │
│                                                              │
│  4. Network Error                                           │
│     → Retry connection                                      │
│     → Show error message to user                            │
│                                                              │
│  5. Backend Unavailable                                     │
│     → Queue requests locally                                │
│     → Sync when connection restored                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Key Benefits

```
┌──────────────────────────────────────────────────────────┐
│                    SECURITY                               │
├──────────────────────────────────────────────────────────┤
│  ✓ Short-lived tokens (5 min expiry)                     │
│  ✓ One-time use sessions                                 │
│  ✓ No persistent authentication on station               │
│  ✓ Replay attack prevention                              │
│  ✓ Session isolation per user                            │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│                   USER EXPERIENCE                         │
├──────────────────────────────────────────────────────────┤
│  ✓ Quick QR scan to start                                │
│  ✓ Visual session status feedback                        │
│  ✓ Secure and private transactions                       │
│  ✓ Explicit session control                              │
│  ✓ Clear start/end boundaries                            │
└──────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│                  SYSTEM DESIGN                            │
├──────────────────────────────────────────────────────────┤
│  ✓ Stateless station app                                 │
│  ✓ Scalable session management                           │
│  ✓ Easy multi-station support                            │
│  ✓ Clear audit trail                                     │
│  ✓ Maintainable architecture                             │
└──────────────────────────────────────────────────────────┘
```
