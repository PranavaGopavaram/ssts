-- SSTS Database Initialization
-- This script sets up the initial database schema and data

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create indexes for better performance
-- These will be created by GORM automatically, but we can define additional ones here

-- Create initial admin user (if authentication is enabled)
-- This would be handled by the application