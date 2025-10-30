DROP TABLE IF EXISTS coordinates;
DROP TABLE IF EXISTS drivers_sessions;
DROP TABLE IF EXISTS ride_events;
DROP TABLE IF EXISTS location_history;

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS drivers;
DROP TABLE IF EXISTS rides;


-- User roles enumeration
CREATE TYPE roles AS ENUM (
  'PASSENGER', -- Passenger/Customer
  'ADMIN' -- Administrator
);

-- User status enumeration
CREATE TYPE user_status AS ENUM (
  'ACTIVE', -- Active user
  'INACTIVE', -- Inactive/Suspended
  'BANNED' -- Banned user
);

-- Ride status enumeration
CREATE TYPE ride_status AS ENUM (
  'REQUESTED', -- Ride has been requested by customer
  'MATCHED', -- Driver has been matched to the ride
  'EN_ROUTE', -- Driver is on the way to pickup location
  'ARRIVED', -- Driver has arrived at pickup location
  'IN_PROGRESS', -- Ride is currently in progress
  'COMPLETED', -- Ride has been successfully completed
  'CANCELLED' -- Ride was cancelled
);

-- Vehicle type enumeration
CREATE TYPE vehicle_type AS ENUM (
  'ECONOMY', -- Standard economy ride
  'PREMIUM', -- Premium comfort ride
  'XL' -- Extra large vehicle for groups
);

-- Event type enumeration for audit trail
CREATE TYPE ride_event_type AS ENUM (
  'RIDE_REQUESTED', -- Initial ride request
  'DRIVER_MATCHED', -- Driver assigned to ride
  'DRIVER_ARRIVED', -- Driver arrived at pickup
  'RIDE_STARTED', -- Ride began
  'RIDE_COMPLETED', -- Ride finished
  'RIDE_CANCELLED', -- Ride was cancelled
  'STATUS_CHANGED', -- General status change
  'LOCATION_UPDATED', -- Location update during ride
  'FARE_ADJUSTED' -- Fare was adjusted
);

-- Driver status enumeration
CREATE TYPE driver_status AS ENUM (
  'OFFLINE', -- Driver is not accepting rides
  'AVAILABLE', -- Driver is available to accept rides
  'BUSY', -- Driver is currently occupied
  'EN_ROUTE' -- Driver is on the way to pickup
);


CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE EXTENSION IF NOT EXISTS postgis_topology;



CREATE TABLE IF NOT EXISTS users (
                                     user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role roles DEFAULT 'PASSENGER',
    status user_status DEFAULT 'ACTIVE',
    attrs JSONB DEFAULT '{}'::JSONB
    );

CREATE TABLE IF NOT EXISTS coordinates (
                                           coord_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    entity_id UUID NOT NULL, -- driver_id or passenger_id
    entity_type TEXT NOT NULL CHECK (entity_type in ('DRIVER', 'PASSENGER')),
    address TEXT NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude DECIMAL(11, 8) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    fare_amount DECIMAL(10, 2) CHECK (fare_amount >= 0),
    distance_km DECIMAL(8, 2) CHECK (distance_km >= 0),
    duration_minutes INTEGER CHECK (duration_minutes >= 0),
    is_current BOOLEAN DEFAULT true
    );

CREATE TABLE IF NOT EXISTS drivers (
                                       driver_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    license_number TEXT UNIQUE NOT NULL,
    vehicle_type vehicle_type NOT NULL,
    vehicle_attrs JSONB DEFAULT '{}'::JSONB,
    rating DECIMAL(3,2) DEFAULT 5.0 CHECK (rating BETWEEN 1.0 AND 5.0),
    total_rides INTEGER DEFAULT 0 CHECK (total_rides >= 0),
    total_earnings DECIMAL(10,2) DEFAULT 0 CHECK (total_earnings >= 0),
    status driver_status DEFAULT 'OFFLINE',
    is_verified BOOLEAN DEFAULT false
    );


CREATE TABLE IF NOT EXISTS driver_sessions (
                                               driver_session_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    driver_id UUID REFERENCES drivers (driver_id) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    ended_at TIMESTAMPTZ,
    total_rides INTEGER DEFAULT 0,
    total_earnings DECIMAL(10, 2) DEFAULT 0
    );

CREATE TABLE IF NOT EXISTS rides (
                                     ride_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    ride_number TEXT UNIQUE NOT NULL,
    passenger_id UUID NOT NULL REFERENCES users (user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    driver_id UUID REFERENCES drivers (driver_id),
    vehicle_type vehicle_type NOT NULL DEFAULT 'ECONOMY',
    status ride_status DEFAULT 'REQUESTED',
    priority INTEGER DEFAULT 1 CHECK (priority BETWEEN 1 AND 10),
    requested_at TIMESTAMPTZ DEFAULT NOW (),
    matched_at TIMESTAMPTZ,
    arrived_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    estimated_fare DECIMAL(10, 2),
    final_fare DECIMAL(10, 2),
    pickup_coord_id UUID REFERENCES coordinates (coord_id),
    destination_coord_id UUID REFERENCES coordinates (coord_id)
    );


CREATE TABLE IF NOT EXISTS ride_events (
                                           ride_event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
    ride_id UUID REFERENCES rides (ride_id) NOT NULL,
    event_type ride_event_type,
    event_data jsonb NOT NULL
    );


CREATE TABLE IF NOT EXISTS location_history (
                                                location_history_id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    coord_id UUID REFERENCES coordinates (coord_id),
    driver_id UUID REFERENCES drivers (driver_id),
    latitude DECIMAL(10, 8) NOT NULL CHECK (latitude BETWEEN -90 AND 90),
    longitude DECIMAL(13, 8) NOT NULL CHECK (longitude BETWEEN -180 AND 180),
    accuracy_meters DECIMAL(6, 2),
    speed_kmh DECIMAL(5, 2),
    heading_degrees DECIMAL(5, 2) CHECK (heading_degrees BETWEEN 0 AND 360),
    recorded_at timestamptz NOT NULL DEFAULT NOW (),
    ride_id UUID REFERENCES rides (ride_id)
    );

-- Add indexes for better performance
CREATE INDEX IF NOT EXISTS idx_coordinates_entity_current ON coordinates(entity_id, entity_type, is_current);
CREATE INDEX IF NOT EXISTS idx_rides_driver_status ON rides(driver_id, status);
CREATE INDEX IF NOT EXISTS idx_rides_status ON rides(status);
CREATE INDEX IF NOT EXISTS idx_driver_sessions_driver_ended ON driver_sessions(driver_id, ended_at);
CREATE INDEX IF NOT EXISTS idx_location_history_driver_recorded ON location_history(driver_id, recorded_at);

-- Add unique constraint for current driver coordinates
CREATE UNIQUE INDEX IF NOT EXISTS idx_coordinates_driver_current
    ON coordinates(entity_id, entity_type, is_current)
    WHERE is_current = true AND entity_type = 'DRIVER';