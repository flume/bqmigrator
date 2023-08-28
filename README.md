# BQMigrator: BigQuery Migration Tool For Go Projects

BQMigrator is a Go package that simplifies the process of writing and running BigQuery migrations. This tool is designed
to streamline the management of database schema changes and data transformations in BigQuery, making it easier for Go
developers to maintain and evolve their data infrastructure.

## Overview

BQMigrator offers a set of features to facilitate the execution of migrations on BigQuery datasets. The core
functionalities of the package include:

- **Migration Registration**: The package provides a method to register individual migrations. Migrations are organized
  based on sequential numbers and are executed in order.

- **Migration Execution**: BQMigrator allows you to execute pending migrations on a specified dataset. It automatically
  tracks the latest completed migration and runs only the new ones.

- **Data Safety**: Before applying a migration, BQMigrator creates copies of the relevant tables. In case of migration
  failure, the original state can be restored.

## Installation

To use BQMigrator in your project, install it and its dependencies:

   ```sh
   go get github.com/flume/bqmigrator
   go get cloud.google.com/go/bigquery
   ```

## Usage

To utilize BQMigrator in your project, follow these steps:

1. **Initialization**: Create a `Migrator` instance using the `bqmigrator.New` function.
   You can customize the dataset and table names for migrations. They default to `migrations` and `migrations`
   respectively.
   Be sure to import your migrations directory so that they are registered.

   ```go
   import (
      "context"
      
      "cloud.google.com/go/bigquery"
      
      "github.com/flume/bqmigrator"
      _ "import/path/to/migrations/directory"
   )
    ctx := context.Background()
    bq, err := bigquery.NewClient(ctx, "your-project-id")
    if err != nil {
        fmt.Println(err)
    }
    defer bq.Close()
   
    migrator := bqmigrator.New(
        bq,
        bqmigrator.WithDataset("custom_dataset"),  // default: "migrations"
        bqmigrator.WithTable("custom_table"),  // default: "migrations"
    )
   
    err = migrator.Migrate(ctx)
    if err != nil {
        fmt.Println(err)
    }
   ```

2. **Register Migrations**: Define your migration functions and register them using the `bqmigrator.RegisterMigration`
   method.
   Use the provided types to construct the migration with name, description, target, and functions for setup and run.
   It is a good idea to keep these in the same directory but to follow some naming convention so that the

   ```go
   func init() {
      bqmigrator.Migration{
         Name:        "0001_my_first_migration",
         Description: "This is my first migration",
         Target:      bqmigrator.Target{
            ProjectID: "fluxus-staging",
            Datasets: []bqmigrator.Dataset{
               {
                  Name: "my_dataset",
                  Tables: []string{"table_1", "table_2"},
               },
            },
         },
         Setup: func(ctx context.Context, bqclient *bq.Client, migration *bqmigrator.Migration) error {
            // Do some setup here
            // Notice how the migration is passed in as a pointer, this allows you to dynamically add datasets and tables to the migration
            // This is useful if you want to add a table to the migration based on some condition
            return nil
         },
         Run: func(ctx context.Context, bqclient *bq.Client, migration bqmigrator.Migration) error {
            // Do the migratione here
            // Your bigquery client is passed in so you can run queries, update tables, or change datasets, really anything
            return nil
         },
      }
      err := bqmigrator.RegisterMigration(migration)
      if err != nil {
         panic(fmt.Errorf("registering migration %s: %v", migration.Name, err))
      }
   }
   ```

3. **Run Migrations**: Execute pending migrations using the `Migrate` method. You can run this as apart of a script/cli
   or as part of your application startup.

   ```go
   err := migrator.Migrate(ctx)
   if err != nil {
      fmt.Println(err)
   }
   ```

## Contributing

We welcome contributions to enhance and extend BQMigrator. If you would like to contribute, please follow the guidelines
outlined in the CONTRIBUTING.md file in the repository.

## License

BQMigrator is open-source software licensed under the [MIT License](https://opensource.org/licenses/MIT). Feel free to
use, modify, and distribute it as per the terms of the license.
