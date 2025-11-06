# Ride-Booking-Application
real-time platform connecting passengers with drivers for seamless ride requests, dynamic driver matching, and live tracking. Built with microservices and SOA, it uses RabbitMQ for message queuing and WebSockets for real-time updates, providing a robust, fault-tolerant system for high concurrency
`
## Overview

The Ride-Hail System is a distributed, real-time platform designed to manage the ride-hailing process, from ride requests to driver matching, real-time tracking, and completion. The system is built using Go and follows Service-Oriented Architecture (SOA) principles, utilizing RabbitMQ for messaging, WebSockets for real-time communication, and PostgreSQL for database operations.

This repository contains the core service configurations and implementations that power the platform. The system handles various services such as ride service, driver location service, admin service, and authentication service.

## Project Learning Objectives

- Advanced Message Queue Patterns
- Real-Time Communication (WebSockets)
- Geospatial Data Processing
- Complex Microservices Orchestration
- High-Concurrency Programming
- Distributed State Management
- Service-Oriented Architecture (SOA) Design Patterns

## Table of Contents

- [Overview](#overview)
- [Services](#services)
- [Setup](#setup)
- [Run](#run)
- [Docker Configuration](#docker-configuration)
- [API](#api)
- [Logging and Error Handling](#logging-and-error-handling)
- [System Architecture](#system-architecture)
- [Message Queue Architecture](#message-queue-architecture)
- [Security](#security)

## Services

The Ride-Hail System is composed of the following services:

### 1. Ride Service (ride-service)

Responsible for managing the ride lifecycle, including passenger requests, matching drivers, and handling ride status updates.

### 2. Driver Location Service (driver-location-service)

Handles driver operations, including real-time location tracking, matching drivers to ride requests, and updating driver status.

### 3. Admin Service (admin-service)

Provides monitoring, system analytics, and oversight, allowing administrators to track system health and performance metrics.

### 4. Auth Service (auth-service)

Manages user authentication and authorization, ensuring secure login for both drivers and passengers.

## Setup

### Prerequisites

Ensure you have the following installed:

- Docker (for containerizing and running the services)
- Docker Compose (to orchestrate the services)

### Step 1: Clone the Repository

Clone the repository to your local machine:

```bash
git clone <repository_url>
cd ride-hail
```

### Step 2: Set up Configuration

Ensure that your environment variables are configured. You can modify the `.env` file to include the required variables (or use a `docker-compose.override.yml` for sensitive configurations like database credentials):

```yaml
# Database Configuration
database:
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  user: ${DB_USER:-ridehail_user}
  password: ${DB_PASSWORD:-ridehail_pass}
  database: ${DB_NAME:-ridehail_db}

# RabbitMQ Configuration
rabbitmq:
  host: ${RABBITMQ_HOST:-localhost}
  port: ${RABBITMQ_PORT:-5672}
  user: ${RABBITMQ_USER:-guest}
  password: ${RABBITMQ_PASSWORD:-guest}

# WebSocket Configuration
websocket:
  port: ${WS_PORT:-8080}

# Service Ports
services:
  ride_service: ${RIDE_SERVICE_PORT:-3000}
  driver_location_service: ${DRIVER_LOCATION_SERVICE_PORT:-3001}
  admin_service: ${ADMIN_SERVICE_PORT:-3004}
```

### Step 3: Docker Compose Setup

The system is fully orchestrated using Docker Compose. You can set up and run all services with a single command.

### Step 4: Running the System

Build and start all the services with:

```bash
docker-compose up --build
```

Once the services are up and running, you can access the services locally:

- **Ride Service**: http://localhost:3000
- **Driver Location Service**: http://localhost:3001
- **Admin Service**: http://localhost:3004

### Step 5: Stopping the Services

To stop the services and clean up the containers:

```bash
docker-compose down
```

## Docker Configuration

The project comes with a pre-configured `docker-compose.yml` file, which defines the following services:

- **Ride Service**: Runs on port 3000
- **Driver Location Service**: Runs on port 3001
- **Admin Service**: Runs on port 3004
- **Auth Service**: Runs on port 3002 (for handling user authentication)

### Example docker-compose.yml

```yaml
version: '3.7'

services:
  ride-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - SERVICE_NAME=ride-service
      - DB_HOST=db
      - DB_PORT=5432
      - DB_USER=ridehail_user
      - DB_PASSWORD=ridehail_pass
    depends_on:
      - db
      - rabbitmq

  driver-location-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3001:3001"
    environment:
      - SERVICE_NAME=driver-location-service
      - DB_HOST=db
      - DB_PORT=5432
      - DB_USER=ridehail_user
      - DB_PASSWORD=ridehail_pass
    depends_on:
      - db
      - rabbitmq

  admin-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3004:3004"
    environment:
      - SERVICE_NAME=admin-service
      - DB_HOST=db
      - DB_PORT=5432
      - DB_USER=ridehail_user
      - DB_PASSWORD=ridehail_pass
    depends_on:
      - db
      - rabbitmq

  auth-service:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "3002:3002"
    environment:
      - SERVICE_NAME=auth-service
    depends_on:
      - db

  db:
    image: postgres:13
    environment:
      POSTGRES_USER: ridehail_user
      POSTGRES_PASSWORD: ridehail_pass
      POSTGRES_DB: ridehail_db
    ports:
      - "5432:5432"

  rabbitmq:
    image: rabbitmq:management
    ports:
      - "5672:5672"
      - "15672:15672"
```

## API

### Admin Service

#### System Overview

- **Path**: `/admin/overview`
- **Method**: `GET`
- **Description**: Provides an overview of the system status.

#### Active Rides

- **Path**: `/admin/rides/active`
- **Method**: `GET`
- **Description**: Returns active ride details.

## Logging and Error Handling

Each service follows structured logging with the following mandatory fields:

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | string | ISO 8601 format timestamp |
| `level` | string | INFO, DEBUG, ERROR |
| `service` | string | Service name (e.g., ride-service) |
| `action` | string | Event name (e.g., ride_requested) |
| `message` | string | Human-readable description of the event |
| `hostname` | string | Service hostname |
| `request_id` | string | Correlation ID for tracing |
| `ride_id` | string | Ride identifier (when applicable) |

For **ERROR** logs, the following fields are included:

| Field | Type | Description |
|-------|------|-------------|
| `msg` | string | Error message |
| `stack` | string | Stack trace (if available) |

## System Architecture

The system follows Service-Oriented Architecture (SOA) principles. Key components include:

- **Ride Service**: Manages ride requests and status transitions.
- **Driver & Location Service**: Handles driver matching, status updates, and location tracking.
- **Admin Service**: Provides oversight and real-time analytics for system health.
- **Auth Service**: Manages user authentication and authorization for drivers and passengers.

## Message Queue Architecture

The system uses RabbitMQ to facilitate communication between services:

- **Exchanges**: `ride_topic`, `driver_topic`, `location_fanout`
- **Queues**: Handles ride requests, driver responses, location updates, etc.

## Security

- **JWT Authentication**: Secure all API calls with JWT tokens.
- **Role-Based Access Control (RBAC)**: Implement permissions based on user roles (admin, driver, passenger).
- **Data Protection**: Encrypt sensitive data both at rest and during transit. Use TLS for secure communication.

## Conclusion

The Ride-Hail System demonstrates a robust distributed architecture built to handle real-time ride-hailing tasks, from request to completion. Using Go, RabbitMQ, and PostgreSQL, the system supports multiple microservices interacting via messaging queues and real-time WebSocket communication.
