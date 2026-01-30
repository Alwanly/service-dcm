-- Database initialization script for Controller service
-- This script creates the necessary tables for the distributed configuration management system

-- Create agents table
-- Stores registered agent information
CREATE TABLE IF NOT EXISTS agents (
    agent_id TEXT PRIMARY KEY NOT NULL,
    startup_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL,
    last_poll_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on last_poll_time for efficient querying
CREATE INDEX IF NOT EXISTS idx_agents_last_poll ON agents(last_poll_time);

-- Create configurations table
-- Stores configuration versions with ETags for versioning
CREATE TABLE IF NOT EXISTS configurations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    etag TEXT NOT NULL UNIQUE,
    config_data TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on etag for efficient lookups
CREATE INDEX IF NOT EXISTS idx_configurations_etag ON configurations(etag);

-- Create index on created_at for efficient sorting
CREATE INDEX IF NOT EXISTS idx_configurations_created_at ON configurations(created_at DESC);

-- Insert default configuration if none exists
INSERT OR IGNORE INTO configurations (id, etag, config_data, created_at)
VALUES (1, '1', '{}', CURRENT_TIMESTAMP);

-- Migration: Add agent_configs table for per-agent authentication and configuration
CREATE TABLE IF NOT EXISTS agent_configs (
    id TEXT PRIMARY KEY,                    -- UUID v7
    agent_name TEXT NOT NULL,               -- Hostname from registration
    api_token TEXT NOT NULL UNIQUE,         -- Bearer token for authentication
    poll_interval_seconds INTEGER,          -- Per-agent interval (NULL = use global default)
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast token lookup during authentication
CREATE INDEX IF NOT EXISTS idx_agent_configs_api_token ON agent_configs(api_token);

-- Index for listing agents by name
CREATE INDEX IF NOT EXISTS idx_agent_configs_agent_name ON agent_configs(agent_name);
