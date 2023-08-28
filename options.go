package bqmigrator

type MigratorOption = func(*Migrator)

func WithDatasetName(dataset string) MigratorOption {
	return func(m *Migrator) {
		m.dataset = dataset
	}
}

func WithTableName(table string) MigratorOption {
	return func(m *Migrator) {
		m.table = table
	}
}
