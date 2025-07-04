-- PostgreSQL initialization script for St. Planer

-- Create database if not exists (this is handled by docker-compose POSTGRES_DB)
-- Just add any initial setup here if needed

-- Grant all privileges to the stplaner user
GRANT ALL PRIVILEGES ON DATABASE stplaner TO stplaner;

-- Create extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Set default timezone
SET timezone = 'UTC';

-- Add any initial data or additional setup here
-- For example, you might want to create initial categories or templates

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'St. Planer database initialized successfully';
END $$;