package bqmigrator

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/api/googleapi"

	"context"

	"cloud.google.com/go/bigquery"
)

type bqclient struct {
	client *bigquery.Client
}

func newClient(bq *bigquery.Client) *bqclient {
	return &bqclient{
		client: bq,
	}
}

func (bq *bqclient) GetClient() *bigquery.Client {
	return bq.client
}

func (bq *bqclient) Query(ctx context.Context, query string) (RowIterator, error) {
	q := bq.client.Query(query)
	iter, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying: %v", err)
	}
	return &RowIteratorWrapper{RowIterator: iter}, nil
}

func (bq *bqclient) CreateDataset(ctx context.Context, name string) error {
	exists, err := bq.datasetExists(ctx, name)
	if err != nil {
		return fmt.Errorf("checking dataset exists: %v", err)
	}
	if exists {
		return nil
	}

	dataset := bq.client.Dataset(name)
	err = dataset.Create(ctx, &bigquery.DatasetMetadata{Name: name})
	if err != nil {
		if isAlreadyExistsError(err) {
			return nil
		}
		return fmt.Errorf("creating dataset: %v", err)
	}

	exists, err = retryUntil(func() (bool, error) {
		exists, err = bq.datasetExists(ctx, name)
		if err != nil {
			return false, fmt.Errorf("checking dataset exists: %v", err)
		}
		if exists {
			return true, nil
		}
		return false, nil
	}, 12, 5*time.Second, true)

	if err != nil {
		return fmt.Errorf("waiting for dataset to exist: %v", err)
	}
	return nil
}

func (bq *bqclient) datasetExists(ctx context.Context, dataset string) (exists bool, err error) {
	datasetRef := bq.client.Dataset(dataset)
	_, err = datasetRef.Metadata(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("checking dataset metadata: %v", err)
		}
	}
	return true, nil
}

// CreateTable first checks if the table exists, if it does not exist it creates one and waits for it to exists
func (bq *bqclient) CreateTable(ctx context.Context, dataset, table string, schema bigquery.Schema) (exists bool, err error) {
	err = bq.CreateDataset(ctx, dataset)
	if err != nil {
		return false, err
	}

	exists, err = bq.tableExists(ctx, dataset, table)
	if err != nil {
		return false, fmt.Errorf("checking table exists: %v", err)
	}
	if exists {
		return true, nil
	}

	tableRef := bq.client.Dataset(dataset).Table(table)

	if err = tableRef.Create(ctx, &bigquery.TableMetadata{Name: table, Schema: schema}); err != nil {
		return false, fmt.Errorf("creating table: %v", err)
	}

	exists, err = retryUntil(func() (bool, error) {
		exists, err = bq.tableExists(ctx, dataset, table)
		if err != nil {
			return false, fmt.Errorf("checking table exists: %v", err)
		}
		if exists {
			return true, nil
		}
		return false, nil
	}, 12, 5*time.Second, true)

	if err != nil {
		return false, fmt.Errorf("waiting for table to exist: %v", err)
	}
	return exists, nil
}

func (bq *bqclient) tableExists(ctx context.Context, dataset, table string) (created bool, err error) {
	tableRef := bq.client.Dataset(dataset).Table(table)
	_, err = tableRef.Metadata(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("checking table metadata: %v", err)
		}
	}
	return true, nil
}

func (bq *bqclient) CreateView(ctx context.Context, dataset, view, query string) error {
	exists, err := bq.tableExists(ctx, dataset, view)
	if err != nil {
		return fmt.Errorf("checking view exists: %v", err)
	}
	if exists {
		return nil
	}

	q := bq.client.Query(query)
	_, err = q.Read(ctx)
	if err != nil {
		return fmt.Errorf("creating view: %v", err)
	}
	return nil
}

func (bq *bqclient) CreateTableFunction(ctx context.Context, dataset, function, query string) error {
	routine := bq.client.Dataset(dataset).Routine(function)
	_, err := routine.Metadata(ctx)
	if err != nil {
		if isNotFoundErr(err) {
			q := bq.client.Query(query)
			_, err = q.Read(ctx)
			if err != nil {
				return fmt.Errorf("creating table function: %v", err)
			}
		} else {
			return fmt.Errorf("checking routine metadata: %v", err)
		}
	}
	return nil
}

func (bq *bqclient) CopyTable(ctx context.Context, dataset, table, copy string) error {
	copier := bq.client.Dataset(dataset).Table(copy).CopierFrom(bq.client.Dataset(dataset).Table(table))
	copier.WriteDisposition = bigquery.WriteTruncate
	job, err := copier.Run(ctx)
	if err != nil {
		return fmt.Errorf("copying table: %v", err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("waiting for copy job to complete: %v", err)
	}
	if err = status.Err(); err != nil {
		return fmt.Errorf("copy job failed: %v", err)
	}
	return nil
}

func (bq *bqclient) DeleteTable(ctx context.Context, dataset, table string) error {
	err := bq.client.Dataset(dataset).Table(table).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting table: %v", err)
	}
	return nil
}

func isNotFoundErr(err error) bool {
	var e *googleapi.Error
	ok := errors.As(err, &e)
	return ok && e.Code == http.StatusNotFound
}

func isAlreadyExistsError(err error) bool {
	var e *googleapi.Error
	if errors.As(err, &e) {
		if e.Code == 409 || (e.Code == 400 && strings.Contains(e.Message, "already exists in schema")) {
			return true
		}
	}
	return false
}

type RowIteratorWrapper struct {
	RowIterator *bigquery.RowIterator
}

func (r *RowIteratorWrapper) Next(dst any) error {
	return r.RowIterator.Next(dst)
}

func (r *RowIteratorWrapper) TotalRows() int {
	return int(r.RowIterator.TotalRows)
}

type RowIterator interface {
	Next(dst any) error
	TotalRows() int
}
