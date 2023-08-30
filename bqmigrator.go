package bqmigrator

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type bq interface {
	GetClient() *bigquery.Client
	Query(ctx context.Context, query string) (rowIterator, error)
	CreateDataset(ctx context.Context, dataset string) error
	CreateTable(ctx context.Context, dataset, table string, schema bigquery.Schema) (exists bool, err error)
	CopyTable(ctx context.Context, dataset, table, copy string) error
	DeleteTable(ctx context.Context, dataset, table string) error
}

var (
	migrations = map[int]Migration{}
)

func RegisterMigration(m Migration) error {
	num, err := parseMigrationNumber(m.Name)
	if err != nil {
		return fmt.Errorf("parsing migration number: %v", err)
	}

	if _, ok := migrations[num]; ok {
		return fmt.Errorf("migration %d already exists", num)
	}

	m.number = num
	migrations[num] = m
	return nil
}

type Migrator struct {
	dataset string
	table   string
	bq      bq
}

func New(bq *bigquery.Client, opts ...MigratorOption) *Migrator {
	migrator := &Migrator{
		dataset: "migrations",
		table:   "migrations",
		bq:      newClient(bq),
	}

	for _, opt := range opts {
		opt(migrator)
	}

	return migrator
}

func (m *Migrator) Migrate(ctx context.Context) (err error) {
	// Ensure dataset and Tables exist
	fmt.Println("Setting up bigquery")
	err = m.setupBigquery(ctx)
	if err != nil {
		return fmt.Errorf("setting up bigquery: %v", err)
	}

	latest, err := m.getLatestMigrationNumber(ctx)
	if err != nil {
		return fmt.Errorf("getting latest migration number: %v", err)
	}
	fmt.Printf("Last migration number: %d\n", latest)

	for _, migration := range m.getOrderedMigrations() {
		if migration.number <= latest {
			fmt.Printf("%s already run\n", migration.Name)
			continue
		}
		if migration.Run == nil {
			return fmt.Errorf("migration %s has no run function", migration.Name)
		}
		err = m.runMigration(ctx, migration)
		if err != nil {
			return fmt.Errorf("running migration %s: %v", migration.Name, err)
		}
	}
	return nil
}

func (m *Migrator) getOrderedMigrations() []Migration {
	var keys []int
	for k := range migrations {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var orderedMigrations []Migration
	for _, k := range keys {
		orderedMigrations = append(orderedMigrations, migrations[k])
	}
	return orderedMigrations
}

func (m *Migrator) runMigration(ctx context.Context, migration Migration) (err error) {
	if migration.Setup != nil {
		// Run migration setup
		fmt.Printf("Running migration setup: %s\n", migration.Name)
		err = migration.Setup(ctx, m.bq.GetClient(), &migration)
		if err != nil {
			return fmt.Errorf("running migration setup: %v", err)
		}
	} else {
		fmt.Printf("Skipping migration setup: %s\n", migration.Name)
	}

	// Copy tables in case something goes wrong
	fmt.Println("Creating copies of tables")
	err = m.copyTables(ctx, migration)
	if err != nil {
		return fmt.Errorf("copying tables: %v", err)
	}

	// Revert tables if there is an error after this point
	defer func(ctx context.Context, migration Migration) {
		if err != nil {
			fmt.Println("Reverting tables to original state")
			rerr := m.revertDatasets(ctx, migration)
			if rerr != nil {
				err = fmt.Errorf("reverting tables: %v", errors.Join(rerr, err))
			}
		}
	}(ctx, migration)

	// Run migration
	fmt.Printf("Running migration: %s\n", migration.Name)
	err = migration.Run(ctx, m.bq.GetClient(), migration)
	if err != nil {
		return fmt.Errorf("running migration: %v", err)
	}

	// Insert migration
	fmt.Println("Inserting migration into migrations tables")
	err = m.insertMigrations(ctx, migration)
	if err != nil {
		return fmt.Errorf("inserting migrations: %v", err)
	}

	// Delete copied datasets
	fmt.Println("Deleting copied tables")
	err = m.deleteCopiedTables(ctx, migration)
	if err != nil {
		return fmt.Errorf("deleting copied tables: %v", err)
	}

	fmt.Printf("Completed migration: %s\n", migration.Name)
	return nil
}

func (m *Migrator) setupBigquery(ctx context.Context) error {
	// Create dataset if it doesn't exist
	err := m.bq.CreateDataset(ctx, m.dataset)
	if err != nil {
		return fmt.Errorf("creating dataset: %v", err)
	}

	exists, cerr := m.bq.CreateTable(ctx, m.dataset, m.table, migrationTableSchema)
	if cerr != nil {
		return fmt.Errorf("creating table: %v", cerr)
	}
	if !exists {
		return fmt.Errorf("table %v.%v does not exist", m.dataset, m.table)
	}
	return nil
}

func (m *Migrator) getLatestMigrationNumber(ctx context.Context) (int, error) {
	// Create query to get the latest migration number
	query := fmt.Sprintf(migrationNumberQuery, fullTableName(m.dataset, m.table))

	// Run query
	iter, err := m.bq.Query(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("querying for migration number: %v", err)
	}

	var latestMigrationNumber int
	for {
		var latest latestMigration
		nerr := iter.Next(&latest)
		if nerr != nil {
			if errors.Is(nerr, iterator.Done) {
				break
			}
			return 0, fmt.Errorf("iterating over latest migration number: %v", nerr)
		}

		if iter.TotalRows() != 1 {
			return 0, fmt.Errorf("expected 1 row, got %d", iter.TotalRows())
		}

		latestMigrationNumber = latest.migrationNumber
	}
	return latestMigrationNumber, nil
}

func (m *Migrator) copyTables(ctx context.Context, migration Migration) error {
	for _, dataset := range migration.Target.Datasets {
		for _, table := range dataset.Tables {
			err := m.bq.CopyTable(ctx, dataset.Name, table, tableCopyName(table))
			if err != nil {
				return fmt.Errorf("copying table: %v", err)
			}
		}
	}
	return nil
}

func (m *Migrator) revertDatasets(ctx context.Context, migration Migration) error {
	for _, dataset := range migration.Target.Datasets {
		for _, table := range dataset.Tables {
			err := m.bq.CopyTable(ctx, dataset.Name, tableCopyName(table), table)
			if err != nil {
				return fmt.Errorf("copying table: %v", err)
			}
		}
	}

	err := m.deleteCopiedTables(ctx, migration)
	if err != nil {
		return fmt.Errorf("deleting copied tables: %v", err)
	}
	return nil
}

func (m *Migrator) deleteCopiedTables(ctx context.Context, migration Migration) error {
	for _, dataset := range migration.Target.Datasets {
		for _, table := range dataset.Tables {
			err := m.bq.DeleteTable(ctx, dataset.Name, tableCopyName(table))
			if err != nil {
				return fmt.Errorf("deleting table: %v", err)
			}
		}
	}
	return nil
}

func (m *Migrator) insertMigrations(ctx context.Context, migration Migration) error {
	query := fmt.Sprintf(
		migrationInsertQuery,
		fullTableName(m.dataset, m.table),
		migration.Name,
		migration.Description,
		getDatasetString(migration.Target),
	)

	_, err := m.bq.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("inserting migration: %v", err)
	}

	return nil
}
