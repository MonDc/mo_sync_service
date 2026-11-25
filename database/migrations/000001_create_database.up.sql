-- =====================================================
-- CREATE DATABASE: mo_db
-- =====================================================
-- This is the primary database for the MO User Sync Service.
-- It stores synced user data and service-specific tables.
-- =====================================================

CREATE DATABASE IF NOT EXISTS mo_db
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

-- =====================================================
-- After creation, switch to the database and verify:
-- USE mo_db;
-- SHOW TABLES;
-- =====================================================
