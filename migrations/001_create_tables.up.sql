BEGIN;

-- Create roles and user status tables if they do not exist
create table if not exists "roles"("value" text not null primary key);
insert into "roles" ("value")
values
    ('PASSENGER'),
    ('DRIVER'),
    ('ADMIN')
    on conflict do nothing;

create table if not exists "user_status"("value" text not null primary key);
insert into "user_status" ("value")
values
    ('ACTIVE'),
    ('INACTIVE'),
    ('BANNED')
    on conflict do nothing;

-- Main users table creation
create table if not exists users (
                                     id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    email varchar(100) unique not null,
    role text references "roles"(value) not null,
    status text references "user_status"(value) not null default 'ACTIVE',
    password_hash text not null,
    attrs jsonb default '{}'::jsonb
    );

-- Ride status enumeration
create table if not exists "ride_status"("value" text not null primary key);
insert into "ride_status" ("value")
values
    ('REQUESTED'),
    ('MATCHED'),
    ('EN_ROUTE'),
    ('ARRIVED'),
    ('IN_PROGRESS'),
    ('COMPLETED'),
    ('CANCELLED')
    on conflict do nothing;

-- Vehicle type enumeration
create table if not exists "vehicle_type"("value" text not null primary key);
insert into "vehicle_type" ("value")
values
    ('ECONOMY'),
    ('PREMIUM'),
    ('XL')
    on conflict do nothing;

-- Coordinates table for real-time location tracking
create table if not exists coordinates (
                                           id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    entity_id uuid not null,
    entity_type varchar(20) not null check (entity_type in ('driver', 'passenger')),
    address text not null,
    latitude decimal(10,8) not null check (latitude between -90 and 90),
    longitude decimal(11,8) not null check (longitude between -180 and 180),
    fare_amount decimal(10,2) check (fare_amount >= 0),
    distance_km decimal(8,2) check (distance_km >= 0),
    duration_minutes integer check (duration_minutes >= 0),
    is_current boolean default true
    );

-- Create indexes for coordinates
create index if not exists idx_coordinates_entity on coordinates(entity_id, entity_type);
create index if not exists idx_coordinates_current on coordinates(entity_id, entity_type) where is_current = true;

-- Main rides table
create table if not exists rides (
                                     id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    ride_number varchar(50) unique not null,
    passenger_id uuid not null references users(id),
    driver_id uuid references users(id),
    vehicle_type text references "vehicle_type"(value),
    status text references "ride_status"(value),
    priority integer default 1 check (priority between 1 and 10),
    requested_at timestamptz default now(),
    matched_at timestamptz,
    arrived_at timestamptz,
    started_at timestamptz,
    completed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text,
    estimated_fare decimal(10,2),
    final_fare decimal(10,2),
    pickup_coordinate_id uuid references coordinates(id),
    destination_coordinate_id uuid references coordinates(id)
    );

-- Create index for rides status
create index if not exists idx_rides_status on rides(status);

-- Driver status enumeration
create table if not exists "driver_status"("value" text not null primary key);
insert into "driver_status" ("value")
values
    ('AVAILABLE'),
    ('BUSY'),
    ('OFFLINE'),
    ('BANNED')
    on conflict do nothing;

-- Driver sessions table
create table if not exists driver_sessions (
                                               id uuid primary key default gen_random_uuid(),
    driver_id uuid not null references users(id),
    started_at timestamptz not null default now(),
    ended_at timestamptz,
    total_rides integer default 0,
    total_earnings decimal(10,2) default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
    );

-- Location history table for real-time tracking
create table if not exists location_history (
                                                id uuid primary key default gen_random_uuid(),
    coordinate_id uuid references coordinates(id),
    driver_id uuid not null references users(id),
    latitude decimal(10,8) not null check (latitude between -90 and 90),
    longitude decimal(11,8) not null check (longitude between -180 and 180),
    accuracy_meters decimal(5,2),
    speed_kmh decimal(5,2),
    heading_degrees decimal(5,2),
    ride_id uuid references rides(id),
    created_at timestamptz not null default now()
    );

-- Drivers table extending users
create table if not exists drivers (
                                       id uuid primary key references users(id),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    license_number varchar(50) unique not null,
    vehicle_make varchar(50) not null,
    vehicle_model varchar(50) not null,
    vehicle_year integer not null,
    vehicle_color varchar(30) not null,
    license_plate varchar(20) not null,
    vehicle_type text references "vehicle_type"(value) not null,
    status text references "driver_status"(value) not null default 'OFFLINE',
    rating decimal(2,1) default 5.0 check (rating between 0 and 5),
    total_rides integer default 0,
    total_earnings decimal(12,2) default 0,
    is_verified boolean default false,
    current_ride_id uuid references rides(id)
    );

-- Create indexes for drivers
create index if not exists idx_drivers_status on drivers(status);
create index if not exists idx_drivers_vehicle_type on drivers(vehicle_type);
create index if not exists idx_drivers_rating on drivers(rating);

-- Enable PostGIS extension for geospatial queries
-- create extension if not exists postgis;

-- Create spatial index for coordinates
-- create index if not exists idx_coordinates_geom on coordinates using gist (
--     ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
--     );

-- -- Create indexes for location history
-- create index if not exists idx_location_history_geom on location_history using gist (
--     ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
--     );

commit;
