# 🚗 Ride-Hail API Testing Guide

This guide will help you test the Ride-Hail application using Postman. The system provides JWT-based authentication and ride management functionality.

## 📋 Prerequisites

- **Postman** installed on your machine
- **Ride-Hail application** running on `http://localhost:3000`
- **Database and RabbitMQ** services running

## 🏗️ Postman Setup

### 1. Create Collection
- Open Postman
- Click **Collections** → **New Collection**
- Name: `Ride-Hail API`

### 2. Set Up Environment (Recommended)
- Click **Environments** → **New Environment**
- Name: `Ride-Hail Local`
- Add variables:
  - `base_url`: `http://localhost:3000`
  - `token`: (leave empty - will be auto-filled)
  - `user_id`: (leave empty - will be auto-filled)
  - `ride_id`: (leave empty - will be auto-filled)

## 🔐 Authentication Flow

### 1. Register User
**Endpoint:** `POST {{base_url}}/auth/register`

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "email": "passenger@example.com",
  "password": "secret123",
  "name": "John Passenger",
  "phone": "+1234567890",
  "role": "PASSENGER"
}
```

**Tests Tab (Auto-save token):**
```javascript
if (pm.response.code === 201) {
    const response = pm.response.json();
    pm.environment.set("token", response.token);
    pm.environment.set("user_id", response.user.id);
    console.log("Token saved:", response.token);
}
```

**Expected Response (201):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-uuid-here",
    "email": "passenger@example.com",
    "role": "PASSENGER",
    "status": "ACTIVE",
    "name": "John Passenger",
    "phone": "+1234567890",
    "created_at": "2024-12-16T10:30:00Z"
  }
}
```

### 2. Login User
**Endpoint:** `POST {{base_url}}/auth/login`

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "email": "passenger@example.com",
  "password": "secret123"
}
```

**Tests Tab:**
```javascript
if (pm.response.code === 200) {
    const response = pm.response.json();
    pm.environment.set("token", response.token);
    console.log("Login successful");
}
```

## 👤 Profile Management

### 3. Get User Profile
**Endpoint:** `GET {{base_url}}/auth/profile`

**Headers:**
```
Authorization: Bearer {{token}}
```

**Expected Response (200):**
```json
{
  "id": "user-uuid-here",
  "email": "passenger@example.com",
  "role": "PASSENGER",
  "status": "ACTIVE",
  "name": "John Passenger",
  "phone": "+1234567890",
  "created_at": "2024-12-16T10:30:00Z"
}
```

## 🚗 Ride Management

### 4. Create Ride
**Endpoint:** `POST {{base_url}}/rides`

**Headers:**
```
Content-Type: application/json
Authorization: Bearer {{token}}
```

**Body:**
```json
{
  "passenger_id" : "
  "pickup_latitude": 43.238949,
  "pickup_longitude": 76.889709,
  "pickup_address": "Almaty Central Park",
  "destination_latitude": 43.222015,
  "destination_longitude": 76.851511,
  "destination_address": "Kok-Tobe Hill",
  "ride_type": "ECONOMY"
}
```

**Tests Tab:**
```javascript
if (pm.response.code === 201) {
    const response = pm.response.json();
    pm.environment.set("ride_id", response.ride_id);
    console.log("Ride created:", response.ride_id);
}
```

**Expected Response (201):**
```json
{
  "ride_id": "ride-uuid-here",
  "ride_number": "RIDE_20241216_001",
  "status": "REQUESTED",
  "estimated_fare": 1450.0,
  "estimated_duration_minutes": 15,
  "estimated_distance_km": 5.2
}
```

### 5. Cancel Ride
**Endpoint:** `POST {{base_url}}/rides/{{ride_id}}/cancel`

**Headers:**
```
Content-Type: application/json
Authorization: Bearer {{token}}
```

**Body:**
```json
{
  "reason": "Changed my mind"
}
```

**Expected Response (200):**
```json
{
  "ride_id": "ride-uuid-here",
  "status": "CANCELLED",
  "cancelled_at": null,
  "message": "Ride cancelled successfully"
}
```

## 🧪 Error Testing

### 6. Invalid Token Test
**Endpoint:** `GET {{base_url}}/auth/profile`

**Headers:**
```
Authorization: Bearer invalid_token_here
```

**Expected Response (401):** `Invalid token`

### 7. Missing Authentication Test
**Endpoint:** `POST {{base_url}}/rides`

**Headers:**
```
Content-Type: application/json
```

**Body:** (Same as Create Ride)

**Expected Response (401):** `Authorization header required`

### 8. Invalid Vehicle Type Test
**Endpoint:** `POST {{base_url}}/rides`

**Headers:**
```
Content-Type: application/json
Authorization: Bearer {{token}}
```

**Body:**
```json
{
  "pickup_latitude": 43.238949,
  "pickup_longitude": 76.889709,
  "pickup_address": "Almaty Central Park",
  "destination_latitude": 43.222015,
  "destination_longitude": 76.851511,
  "destination_address": "Kok-Tobe Hill",
  "vehicle_type": "INVALID_TYPE"
}
```

**Expected Response (400):** `Invalid vehicle type`

## 📁 Collection Structure

```
Ride-Hail API/
├── Authentication/
│   ├── 01. Register User
│   ├── 02. Login User
│   └── 03. Get Profile
├── Rides/
│   ├── 04. Create Ride
│   └── 05. Cancel Ride
└── Error Testing/
    ├── 06. Invalid Token
    ├── 07. Missing Auth
    └── 08. Invalid Vehicle Type
```

## 🔄 Testing Workflow

1. **Start Fresh**: Use Registration endpoint to create a new user
2. **Verify Authentication**: Test Login and Get Profile endpoints
3. **Test Core Features**: Create and cancel rides
4. **Validate Errors**: Ensure proper error handling
5. **Use Environment**: All tokens and IDs are automatically saved

## ✅ Success Indicators

- ✅ **Green status codes** (200, 201)
- ✅ **JWT token received** and automatically saved
- ✅ **User profile data** returned correctly
- ✅ **Ride creation** successful with ride ID
- ✅ **Proper error messages** for invalid requests

## 🚀 Quick Start with cURL

```bash
# Register new user
curl -X POST http://localhost:3000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "secret123",
    "name": "Test User",
    "phone": "+1234567890",
    "role": "PASSENGER"
  }'

# Get profile with token
curl -X GET http://localhost:3000/auth/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE"
```

## 📊 Monitoring Tips

- **Check Response Status**: Should be 200/201 for success cases
- **Response Time**: Should be under 1 second for local development
- **Postman Console**: Use View → Show Postman Console for detailed logs
- **Environment Variables**: Verify tokens are being saved automatically

---

**Note**: Make sure your Ride-Hail application, database, and RabbitMQ are running before testing. The system uses JWT tokens for authentication, which are automatically managed by the Postman environment.

Happy Testing! 🎯