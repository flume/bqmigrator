package bqmigrator

const (
	migrationNumberQuery = `
SELECT 
  COALESCE(CAST(LTRIM(SUBSTRING(MAX_BY(Name, timestamp), 0, 4), '0') AS INT64), 0) migration_number
FROM %s
`
	migrationInsertQuery = `
INSERT INTO %s (name, description, timestamp, datasets) VALUES ('%s', '%s', CURRENT_TIMESTAMP, %s)
`
)
