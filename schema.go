package bqmigrator

import "cloud.google.com/go/bigquery"

var migrationTableSchema = bigquery.Schema{
	{
		Name:     "name",
		Required: true,
		Type:     bigquery.StringFieldType,
	},
	{
		Name:     "description",
		Required: true,
		Type:     bigquery.StringFieldType,
	},
	{
		Name:     "timestamp",
		Required: true,
		Type:     bigquery.TimestampFieldType,
	},
	{
		Name:     "datasets",
		Repeated: true,
		Type:     bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{
				Name:     "dataset",
				Type:     bigquery.StringFieldType,
				Required: true,
			},
			{
				Name:     "tables",
				Type:     bigquery.StringFieldType,
				Required: true,
				Repeated: true,
			},
		},
	},
}
