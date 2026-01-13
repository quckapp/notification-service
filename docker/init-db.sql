-- QuickChat Notification Service Database Initialization

-- Create database
CREATE DATABASE IF NOT EXISTS quickchat_notification
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE quickchat_notification;

-- Grant privileges
GRANT ALL PRIVILEGES ON quickchat_notification.* TO 'quickchat'@'%';
FLUSH PRIVILEGES;

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    device_token VARCHAR(512) NOT NULL,
    platform ENUM('ios', 'android', 'web') NOT NULL,
    device_model VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    push_enabled BOOLEAN DEFAULT TRUE,
    last_active_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_device_token (device_token),
    INDEX idx_user_id (user_id),
    INDEX idx_user_platform (user_id, platform),
    INDEX idx_last_active (last_active_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    in_app_enabled BOOLEAN DEFAULT TRUE,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start VARCHAR(5) DEFAULT '22:00',
    quiet_hours_end VARCHAR(5) DEFAULT '08:00',
    timezone VARCHAR(50) DEFAULT 'UTC',
    category_settings JSON,
    email_digest_enabled BOOLEAN DEFAULT FALSE,
    email_digest_frequency ENUM('daily', 'weekly') DEFAULT 'daily',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Email templates table (optional)
CREATE TABLE IF NOT EXISTS email_templates (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    html_body TEXT,
    text_body TEXT,
    variables JSON,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default email templates
INSERT INTO email_templates (name, subject, html_body, text_body, variables) VALUES
('welcome', 'Welcome to QuickChat!', '<h1>Welcome {{.Name}}!</h1><p>Thanks for joining QuickChat.</p>', 'Welcome {{.Name}}! Thanks for joining QuickChat.', '["Name"]'),
('password_reset', 'Reset Your Password', '<h1>Password Reset</h1><p>Click <a href="{{.ResetLink}}">here</a> to reset your password.</p>', 'Reset your password: {{.ResetLink}}', '["ResetLink"]'),
('mention', 'You were mentioned in QuickChat', '<p>{{.MentionerName}} mentioned you: "{{.MessagePreview}}"</p>', '{{.MentionerName}} mentioned you: "{{.MessagePreview}}"', '["MentionerName", "MessagePreview"]');

-- Create indexes for common queries
CREATE INDEX idx_email_templates_active ON email_templates(is_active);
