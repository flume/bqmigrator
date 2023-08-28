package bqmigrator

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
)

type Dataset struct {
	Name   string
	Tables []string
}

type Target struct {
	ProjectID string
	Datasets  []Dataset
}

type Migration struct {
	Name        string
	number      int
	Description string
	Target      Target
	Setup       func(ctx context.Context, bigquery *bigquery.Client, migration *Migration) error
	Run         func(ctx context.Context, bigquery *bigquery.Client, migration Migration) error
}

func (d *Dataset) getDatasetString() string {
	var tables string
	for _, table := range d.Tables {
		if tables == "" {
			tables = fmt.Sprintf("'%s'", table)
			continue
		}
		tables = fmt.Sprintf("%s, '%s'", tables, table)
	}
	tableStr := fmt.Sprintf("ARRAY<STRING>[%s]", tables)
	return fmt.Sprintf("STRUCT<STRING, ARRAY<STRING>>('%s', %s)", d.Name, tableStr)
}

type latestMigration struct {
	migrationNumber int `bigquery:"migration_number"`
}

func (c *latestMigration) Load(v []bigquery.Value, s bigquery.Schema) error {
	for i, schema := range s {
		switch schema.Name {
		case "migration_number":
			migrationNumber, ok := v[i].(int64)
			if !ok {
				return fmt.Errorf("migration_number is not an int64")
			}
			c.migrationNumber = int(migrationNumber)
		default:
			return fmt.Errorf("unknown field: [%s]", schema.Name)
		}
	}
	return nil
}
