-- CashbackTV Database Schema
-- Initial migration

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Channels table
CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    source_url TEXT NOT NULL,
    logo JSONB,
    output_config JSONB,
    status VARCHAR(50) NOT NULL DEFAULT 'stopped',
    auto_restart BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_channels_status ON channels(status);

-- Channel logs table (for storing FFmpeg output history)
CREATE TABLE IF NOT EXISTS channel_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id UUID REFERENCES channels(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index on channel_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_channel_logs_channel_id ON channel_logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_logs_created_at ON channel_logs(created_at);

-- System settings table
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default settings
INSERT INTO settings (key, value) VALUES
    ('encoding_presets', '[
        {"name": "High Quality", "preset": "slow", "bitrate": "6000k", "resolution": "1920x1080"},
        {"name": "Standard", "preset": "ultrafast", "bitrate": "4000k", "resolution": "1920x1080"},
        {"name": "Low Bandwidth", "preset": "ultrafast", "bitrate": "2000k", "resolution": "1280x720"}
    ]'::jsonb),
    ('system', '{
        "max_channels": 80,
        "segment_time": 3,
        "playlist_size": 6,
        "log_retention": 1,
        "default_preset": "veryfast",
        "default_bitrate": "3500k",
        "default_resolution": "1920x1080",
        "default_profile": "high",
        "default_crf": 23,
        "default_maxrate": "3800k",
        "default_bufsize": "7600k",
        "auto_restart_enabled": true,
        "use_ramdisk": true,
        "threads_per_process": 1
    }'::jsonb)
ON CONFLICT (key) DO NOTHING;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_channels_updated_at BEFORE UPDATE ON channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_settings_updated_at BEFORE UPDATE ON settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

