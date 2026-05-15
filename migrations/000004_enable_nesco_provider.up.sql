UPDATE providers
SET enabled = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE code = 'nesco';