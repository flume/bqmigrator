package bqmigrator

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseMigrationNumber(migrationName string) (int, error) {
	if len(migrationName) < 4 {
		return 0, errors.New("migration name must be at least 4 characters long")
	}

	matches := regexp.MustCompile(`^\d{4}(_[a-z]+)+$`).MatchString(migrationName)
	if !matches {
		return 0, errors.New("migration name must be in the format 0000_migration_name, regex: ^\\d{4}(_[a-z]+)+$")
	}

	num, err := strconv.Atoi(strings.TrimLeft(migrationName[0:4], "0"))
	if err != nil {
		return 0, fmt.Errorf("parsing migration number: %v", err)
	}

	return num, nil
}

func getDatasetString(target Target) string {
	datasets := make([]string, len(target.Datasets))

	for i, dataset := range target.Datasets {
		datasets[i] = dataset.getDatasetString()
	}

	return fmt.Sprintf("[%s]", strings.Join(datasets, ", "))
}

func tableCopyName(dataset string) string {
	return fmt.Sprintf("%s_copy", dataset)
}

func fullTableName(dataset, table string) string {
	return fmt.Sprintf("`%s.%s`", dataset, table)
}

func retryUntil[T comparable](fn func() (T, error), maxTries int, delay time.Duration, resEq T) (T, error) {
	var result T
	var err error

	for i := 0; i < maxTries; i++ {
		result, err = fn()
		if err == nil && result == resEq {
			return result, nil
		}

		time.Sleep(delay)
	}

	return result, err
}
