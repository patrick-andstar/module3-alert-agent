DROP PROCEDURE IF EXISTS dlp_add_column_if_missing;
DROP PROCEDURE IF EXISTS dlp_add_index_if_missing;

DELIMITER $$

CREATE PROCEDURE dlp_add_column_if_missing(
  IN table_name_value VARCHAR(64),
  IN column_name_value VARCHAR(64),
  IN ddl_value TEXT
)
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = table_name_value
      AND column_name = column_name_value
  ) THEN
    SET @ddl = ddl_value;
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END$$

CREATE PROCEDURE dlp_add_index_if_missing(
  IN table_name_value VARCHAR(64),
  IN index_name_value VARCHAR(64),
  IN ddl_value TEXT
)
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = table_name_value
      AND index_name = index_name_value
  ) THEN
    SET @ddl = ddl_value;
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END$$

DELIMITER ;

CALL dlp_add_column_if_missing(
  'alert_logs',
  'agent_verdict',
  'ALTER TABLE alert_logs ADD COLUMN agent_verdict ENUM(''false_positive'',''true_alert'',''uncertain'')'
);
CALL dlp_add_column_if_missing(
  'alert_logs',
  'agent_confidence',
  'ALTER TABLE alert_logs ADD COLUMN agent_confidence DECIMAL(4,3)'
);
CALL dlp_add_column_if_missing(
  'alert_logs',
  'agent_explanation',
  'ALTER TABLE alert_logs ADD COLUMN agent_explanation TEXT'
);
CALL dlp_add_column_if_missing(
  'alert_logs',
  'recall_score',
  'ALTER TABLE alert_logs ADD COLUMN recall_score DECIMAL(4,3) DEFAULT 0'
);
CALL dlp_add_index_if_missing(
  'alert_logs',
  'idx_agent_verdict',
  'ALTER TABLE alert_logs ADD INDEX idx_agent_verdict (agent_verdict)'
);

CALL dlp_add_column_if_missing(
  'false_positive_library',
  'scenario_key',
  'ALTER TABLE false_positive_library ADD COLUMN scenario_key VARCHAR(512) NULL'
);
CALL dlp_add_column_if_missing(
  'false_positive_library',
  'hit_count',
  'ALTER TABLE false_positive_library ADD COLUMN hit_count INT DEFAULT 1'
);
CALL dlp_add_column_if_missing(
  'false_positive_library',
  'last_seen_at',
  'ALTER TABLE false_positive_library ADD COLUMN last_seen_at DATETIME'
);

UPDATE false_positive_library
SET hit_count = 1
WHERE hit_count IS NULL OR hit_count <= 0;

UPDATE false_positive_library
SET last_seen_at = COALESCE(last_seen_at, created_at, NOW())
WHERE last_seen_at IS NULL;

CREATE TEMPORARY TABLE IF NOT EXISTS dlp_fp_keys AS
SELECT
  id,
  CASE
    WHEN CONCAT_WS('|',
      NULLIF(LOWER(TRIM(COALESCE(sensitive_type, ''))), ''),
      NULLIF(LOWER(TRIM(COALESCE(operation, ''))), ''),
      NULLIF(LOWER(TRIM(COALESCE(process_name, ''))), ''),
      NULLIF(LOWER(TRIM(BOTH '/' FROM COALESCE(target, ''))), '')
    ) = '' THEN CONCAT('legacy|', id)
    ELSE CONCAT_WS('|',
      NULLIF(LOWER(TRIM(COALESCE(sensitive_type, ''))), ''),
      NULLIF(LOWER(TRIM(COALESCE(operation, ''))), ''),
      NULLIF(LOWER(TRIM(COALESCE(process_name, ''))), ''),
      NULLIF(LOWER(TRIM(BOTH '/' FROM COALESCE(target, ''))), '')
    )
  END AS base_key,
  ROW_NUMBER() OVER (
    PARTITION BY
      CASE
        WHEN CONCAT_WS('|',
          NULLIF(LOWER(TRIM(COALESCE(sensitive_type, ''))), ''),
          NULLIF(LOWER(TRIM(COALESCE(operation, ''))), ''),
          NULLIF(LOWER(TRIM(COALESCE(process_name, ''))), ''),
          NULLIF(LOWER(TRIM(BOTH '/' FROM COALESCE(target, ''))), '')
        ) = '' THEN CONCAT('legacy|', id)
        ELSE CONCAT_WS('|',
          NULLIF(LOWER(TRIM(COALESCE(sensitive_type, ''))), ''),
          NULLIF(LOWER(TRIM(COALESCE(operation, ''))), ''),
          NULLIF(LOWER(TRIM(COALESCE(process_name, ''))), ''),
          NULLIF(LOWER(TRIM(BOTH '/' FROM COALESCE(target, ''))), '')
        )
      END
    ORDER BY id
  ) AS key_rank
FROM false_positive_library;

UPDATE false_positive_library fp
JOIN dlp_fp_keys keys_for_update ON keys_for_update.id = fp.id
SET fp.scenario_key = CASE
  WHEN keys_for_update.key_rank = 1 THEN keys_for_update.base_key
  ELSE CONCAT(keys_for_update.base_key, '|legacy-', fp.id)
END
WHERE fp.scenario_key IS NULL OR fp.scenario_key = '';

DROP TEMPORARY TABLE IF EXISTS dlp_fp_keys;

ALTER TABLE false_positive_library MODIFY COLUMN scenario_key VARCHAR(512) NOT NULL;
CALL dlp_add_index_if_missing(
  'false_positive_library',
  'scenario_key',
  'ALTER TABLE false_positive_library ADD UNIQUE INDEX scenario_key (scenario_key)'
);
CALL dlp_add_index_if_missing(
  'false_positive_library',
  'idx_last_seen',
  'ALTER TABLE false_positive_library ADD INDEX idx_last_seen (last_seen_at)'
);

DROP PROCEDURE IF EXISTS dlp_add_column_if_missing;
DROP PROCEDURE IF EXISTS dlp_add_index_if_missing;
