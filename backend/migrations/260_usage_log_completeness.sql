-- NULL 表示历史或未观测路径；true 表示仅有部分用量；false 表示已观察完整终止用量。
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS usage_incomplete boolean;
COMMENT ON COLUMN usage_logs.usage_incomplete IS 'NULL: unobserved; true: incomplete observed usage; false: complete observed usage';
