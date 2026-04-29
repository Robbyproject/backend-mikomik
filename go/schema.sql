-- Mikomik Database Schema
-- Run this after installing MariaDB:
--   mysql -u root -p < schema.sql

CREATE DATABASE IF NOT EXISTS mikomik CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE mikomik;

CREATE TABLE IF NOT EXISTS users (
    id         BIGINT       AUTO_INCREMENT PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    avatar     VARCHAR(255) DEFAULT '',
    role       ENUM('user', 'admin') NOT NULL DEFAULT 'user',
    created_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_role (role),
    INDEX idx_created (created_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS comments (
    id          BIGINT       AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    chapter_id  VARCHAR(100) NOT NULL,
    parent_id   BIGINT       NULL DEFAULT NULL,
    text        TEXT         NOT NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_chapter (chapter_id),
    INDEX idx_parent (parent_id),
    FOREIGN KEY (user_id)   REFERENCES users(id)    ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS reactions (
    id          BIGINT       AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    chapter_id  VARCHAR(100) NOT NULL,
    kind        ENUM('like','love','haha','wow') NOT NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_user_chapter (user_id, chapter_id),
    INDEX idx_chapter (chapter_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS reading_history (
    id             BIGINT       AUTO_INCREMENT PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    content_type   VARCHAR(10)  NOT NULL DEFAULT 'manga',
    manga_id       VARCHAR(100) NOT NULL,
    chapter_id     VARCHAR(100) NOT NULL,
    chapter_number FLOAT        NOT NULL DEFAULT 0,
    title          VARCHAR(255) NOT NULL DEFAULT '',
    cover          VARCHAR(500) NOT NULL DEFAULT '',
    read_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_user_content_chapter (user_id, content_type, manga_id, chapter_id),
    INDEX idx_user_content (user_id, content_type),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS bookmarks (
    id           BIGINT       AUTO_INCREMENT PRIMARY KEY,
    user_id      BIGINT       NOT NULL,
    content_type VARCHAR(10)  NOT NULL DEFAULT 'manga',
    manga_id     VARCHAR(100) NOT NULL,
    title        VARCHAR(255) NOT NULL DEFAULT '',
    cover        VARCHAR(500) NOT NULL DEFAULT '',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uk_user_content (user_id, content_type, manga_id),
    INDEX idx_user (user_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS settings (
    key_name VARCHAR(50) PRIMARY KEY,
    value    TEXT        NOT NULL
) ENGINE=InnoDB;

-- Default settings
INSERT IGNORE INTO settings (key_name, value) VALUES 
('site_title', 'MIKOMIK — Baca Manga, Manhwa & Manhua'),
('site_description', 'MIKOMIK - Baca manga, manhwa, dan manhua terbaru secara gratis. Koleksi terlengkap dengan update harian.'),
('site_keywords', 'manga, manhwa, manhua, baca komik, anime'),
('site_favicon', '/favicon.svg');
