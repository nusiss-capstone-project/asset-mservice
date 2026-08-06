-- asset-mservice schema + mock seed data
-- Compatible with MySQL 8+

CREATE TABLE IF NOT EXISTS `assets` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `symbol` varchar(32) NOT NULL,
  `icon_url` varchar(512) DEFAULT NULL,
  `currency` varchar(16) NOT NULL,
  `current_price` varchar(64) NOT NULL,
  `status` varchar(32) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_assets_status` (`status`),
  UNIQUE KEY `uk_assets_symbol_currency` (`symbol`, `currency`) 
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `asset_orders` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `order_no` varchar(64) NOT NULL,
  `asset_id` bigint NOT NULL,
  `quote_id` varchar(128) NOT NULL,
  `unit_price` varchar(64) NOT NULL,
  `quantity` decimal(36,18) NOT NULL,
  `pay_currency` varchar(16) NOT NULL,
  `pay_amount` decimal(36,18) NOT NULL,
  `payment_id` varchar(128) DEFAULT NULL,
  `status` varchar(32) NOT NULL,
  `idempotency_key` varchar(128) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_asset_orders_order_no` (`order_no`),
  UNIQUE KEY `uk_user_idempotency` (`user_id`, `idempotency_key`),
  KEY `idx_asset_orders_user_id` (`user_id`),
  KEY `idx_asset_orders_asset_id` (`asset_id`),
  KEY `idx_asset_orders_payment_id` (`payment_id`),
  KEY `idx_asset_orders_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `user_asset_holdings` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `asset_id` bigint NOT NULL,
  `quantity` decimal(36,18) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_asset` (`user_id`, `asset_id`),
  KEY `idx_user_asset_holdings_asset_id` (`asset_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `account_ledger` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `ledger_no` varchar(64) NOT NULL,
  `user_id` bigint NOT NULL,
  `asset_code` varchar(32) NOT NULL,
  `change_amount` decimal(36,18) NOT NULL,
  `business_type` varchar(32) NOT NULL,
  `business_id` varchar(128) NOT NULL,
  `balance_after` decimal(36,18) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_account_ledger_ledger_no` (`ledger_no`),
  UNIQUE KEY `uk_ledger_biz` (`business_type`, `business_id`, `asset_code`),
  KEY `idx_account_ledger_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Mock assets for local/dev testing (USD quotes)
INSERT INTO `assets` (`id`, `name`, `symbol`, `icon_url`, `currency`, `current_price`, `status`, `created_at`, `updated_at`)
VALUES
  (1, 'Bitcoin',  'BTC',  'https://cdn.jsdelivr.net/gh/atomiclabs/cryptocurrency-icons@1a63530be6e3747115ea170ddb5ccf963c760c8e/128/color/btc.png',  'USD', '68420.50', 'active',   NOW(3), NOW(3)),
  (2, 'Ethereum', 'ETH',  'https://cdn.jsdelivr.net/gh/atomiclabs/cryptocurrency-icons@1a63530be6e3747115ea170ddb5ccf963c760c8e/128/color/eth.png',  'USD', '3456.78',  'active',   NOW(3), NOW(3)),
  (3, 'Solana',   'SOL',  'https://cdn.jsdelivr.net/gh/atomiclabs/cryptocurrency-icons@1a63530be6e3747115ea170ddb5ccf963c760c8e/128/color/sol.png',  'USD', '148.25',   'active',   NOW(3), NOW(3)),
  (4, 'Ripple',   'XRP',  'https://cdn.jsdelivr.net/gh/atomiclabs/cryptocurrency-icons@1a63530be6e3747115ea170ddb5ccf963c760c8e/128/color/xrp.png',  'USD', '0.62',     'active',   NOW(3), NOW(3)),
  (5, 'Dogecoin', 'DOGE', 'https://cdn.jsdelivr.net/gh/atomiclabs/cryptocurrency-icons@1a63530be6e3747115ea170ddb5ccf963c760c8e/128/color/doge.png', 'USD', '0.14',     'inactive', NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `symbol` = VALUES(`symbol`),
  `icon_url` = VALUES(`icon_url`),
  `currency` = VALUES(`currency`),
  `current_price` = VALUES(`current_price`),
  `status` = VALUES(`status`),
  `updated_at` = NOW(3);
