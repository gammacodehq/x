package applymigrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

type migrationRow struct {
	hash      string
	createdAt int64
}

type queryFile struct {
	name      string
	hash      string
	data      string
	createdAt int64
}

// ApplyMigrations ...
func ApplyMigrations(ctx context.Context, db *sql.DB, migrationsFS fs.FS) error {
	if err := createMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	appliedMigrations, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	queryFiles, err := readMigrationFiles(migrationsFS)
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := applyPendingMigrations(ctx, tx, appliedMigrations, queryFiles); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func createMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS __drizzle_migrations (
			id SERIAL PRIMARY KEY,
			hash TEXT NOT NULL,
			created_at NUMERIC
		)
	`)
	return err
}

func getAppliedMigrations(ctx context.Context, db *sql.DB) ([]migrationRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT hash, created_at 
		FROM __drizzle_migrations 
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []migrationRow
	for rows.Next() {
		var row migrationRow
		if err := rows.Scan(&row.hash, &row.createdAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, row)
	}

	return migrations, rows.Err()
}

func readMigrationFiles(migrationsFS fs.FS) ([]queryFile, error) {
	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return nil, err
	}

	var files []queryFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filename := entry.Name()
		parts := strings.Split(filename, "_")
		if len(parts) == 0 {
			continue
		}

		_, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		file, err := migrationsFS.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", filename, err)
		}

		content, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		hash := sha256.Sum256(content)
		hashStr := hex.EncodeToString(hash[:])

		fileInfo, err := entry.Info()
		createdAt := time.Now().UnixMilli()
		if err == nil {
			createdAt = fileInfo.ModTime().UnixMilli()
		}

		files = append(files, queryFile{
			name:      filename,
			hash:      hashStr,
			data:      string(content),
			createdAt: createdAt,
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})

	return files, nil
}

func applyPendingMigrations(ctx context.Context, tx *sql.Tx, appliedMigrations []migrationRow, queryFiles []queryFile) error {
	appliedCount := len(appliedMigrations)

	for i, queryFile := range queryFiles {
		if i < appliedCount {
			if queryFile.hash != appliedMigrations[i].hash {
				return fmt.Errorf("migrations corrupted: hash mismatch for %s", queryFile.name)
			}
			continue
		}

		if _, err := tx.ExecContext(ctx, queryFile.data); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", queryFile.name, err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO __drizzle_migrations (hash, created_at) 
			VALUES (?, ?)
		`, queryFile.hash, queryFile.createdAt); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", queryFile.name, err)
		}
	}

	return nil
}
