UPDATE providers
SET enabled = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE code = 'nesco';