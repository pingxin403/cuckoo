-- Flash Sale System Database Schema
-- Version: V1
-- Description: Create core tables for flash sale (seckill) system

-- 秒杀活动表 (Seckill Activity Table)
CREATE TABLE seckill_activity (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    activity_id VARCHAR(64) NOT NULL UNIQUE,
    sku_id VARCHAR(64) NOT NULL,
    activity_name VARCHAR(256) NOT NULL,
    total_stock INT NOT NULL,
    remaining_stock INT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    purchase_limit INT DEFAULT 1,
    status TINYINT DEFAULT 0,  -- 0:未开始, 1:进行中, 2:已结束
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_sku_id (sku_id),
    INDEX idx_start_time (start_time),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 秒杀订单表 (Seckill Order Table)
CREATE TABLE seckill_order (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id VARCHAR(64) NOT NULL UNIQUE,
    user_id VARCHAR(64) NOT NULL,
    sku_id VARCHAR(64) NOT NULL,
    activity_id VARCHAR(64) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    status TINYINT NOT NULL DEFAULT 0,  -- 0:待支付, 1:已支付, 2:已取消, 3:超时
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    paid_at DATETIME NULL,
    cancelled_at DATETIME NULL,
    source VARCHAR(16),
    trace_id VARCHAR(64),
    INDEX idx_user_id (user_id),
    INDEX idx_sku_id (sku_id),
    INDEX idx_status_created (status, created_at),
    INDEX idx_activity_id (activity_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 库存流水表 (Stock Log Table)
CREATE TABLE stock_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    sku_id VARCHAR(64) NOT NULL,
    order_id VARCHAR(64) NOT NULL,
    operation TINYINT NOT NULL,  -- 1:扣减, 2:回滚
    quantity INT NOT NULL,
    before_stock INT NOT NULL,
    after_stock INT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_sku_id (sku_id),
    INDEX idx_order_id (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 对账记录表 (Reconciliation Log Table)
CREATE TABLE reconciliation_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    sku_id VARCHAR(64) NOT NULL,
    redis_stock INT NOT NULL,
    redis_sold INT NOT NULL,
    mysql_order_count INT NOT NULL,
    discrepancy_count INT DEFAULT 0,
    status TINYINT DEFAULT 0,  -- 0:正常, 1:有差异, 2:已修复
    details JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_sku_id (sku_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
