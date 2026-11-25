-- =====================================================
-- MO_USERS: Tenants (Customers) belonging to a Merchant
-- =====================================================

CREATE TABLE `mo_users` (
    -- Primary identifiers (Mirror mi_users)
    `id` BIGINT UNSIGNED NOT NULL PRIMARY KEY, -- Matches mi_users.id
    `public_id` CHAR(36) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,

    -- Logical Relationships
    `ma_merchant_id` BIGINT UNSIGNED NOT NULL, -- The Merchant (Parent)

    -- Core Identity (Mirrored from MI)
    `email` VARCHAR(254) NOT NULL,
    `first_name` VARCHAR(100) NOT NULL,
    `last_name` VARCHAR(100) NOT NULL,

    -- Business State
    `remaining_mo` DECIMAL(15,2) NOT NULL DEFAULT 0.00, -- Money field
    `has_mawathiq` TINYINT(1) DEFAULT 0,
    `tenant_status` ENUM('active', 'inactive', 'suspended') DEFAULT 'active',
    `settings` JSON,

    -- Original timestamps & audit
    `original_created_at` TIMESTAMP NOT NULL,
    `original_updated_at` TIMESTAMP NOT NULL,

    -- Sync-Specific Fields
    `first_synced_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `last_synced_at` TIMESTAMP NULL,
    `sync_status` ENUM('pending','synced','failed') DEFAULT 'pending',
    `sync_error` TEXT,

    -- Constraints & Indexes
    UNIQUE KEY `uniq_mo_users_public_id` (`public_id`),
    UNIQUE KEY `uniq_mo_users_mi_user_id` (`id`), 
    INDEX `idx_mo_users_merchant` (`ma_merchant_id`), 
    INDEX `idx_mo_users_status` (`tenant_status`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;