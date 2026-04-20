// Command migrate runs TimeTrak SQL migrations.
//
// Migration files live under ./migrations and use the naming convention:
//
//	NNNN_description.up.sql
//	NNNN_description.down.sql
//
// Tracked in the `schema_migrations` table as it applies.
//
// Subcommands:
//
//	up     apply all pending migrations
//	down   roll back the most recently applied migration
//	redo   run down then up on the most recently applied migration
//	status show what is applied
//	seed   populate a demo dataset (dev only)
//	backfill-rate-snapshots
//	       populate time_entries.rate_rule_id / hourly_rate_minor / currency_code
//	       for closed entries missing a snapshot. Accepts --dry-run.
//	check-rate-snapshots
//	       report closed billable entries with a NULL snapshot, per workspace.
//	       Exits non-zero when any offender is found — intended as a deploy gate.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"timetrak/internal/auth"
)

const migrationsDir = "migrations"

var fileRE = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.(up|down)\.sql$`)

type migration struct {
	version int64
	name    string
	up      string
	down    string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate up|down|redo|status|seed|backfill-rate-snapshots [--dry-run]|check-rate-snapshots")
		os.Exit(2)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := ensureSchemaTable(ctx, pool); err != nil {
		log.Fatalf("ensure schema_migrations: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		if err := cmdUp(ctx, pool); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := cmdDown(ctx, pool); err != nil {
			log.Fatal(err)
		}
	case "redo":
		if err := cmdDown(ctx, pool); err != nil {
			log.Fatal(err)
		}
		if err := cmdUp(ctx, pool); err != nil {
			log.Fatal(err)
		}
	case "status":
		if err := cmdStatus(ctx, pool); err != nil {
			log.Fatal(err)
		}
	case "seed":
		if err := cmdSeed(ctx, pool); err != nil {
			log.Fatal(err)
		}
	case "backfill-rate-snapshots":
		if err := cmdBackfillRateSnapshots(ctx, pool, os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "check-rate-snapshots":
		if err := cmdCheckRateSnapshots(ctx, pool); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(2)
	}
}

func ensureSchemaTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	return err
}

func loadMigrations() ([]migration, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}
	byVersion := map[int64]*migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileRE.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		version, _ := strconv.ParseInt(m[1], 10, 64)
		mg, ok := byVersion[version]
		if !ok {
			mg = &migration{version: version, name: m[2]}
			byVersion[version] = mg
		}
		b, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
		if err != nil {
			return nil, err
		}
		if m[3] == "up" {
			mg.up = string(b)
		} else {
			mg.down = string(b)
		}
	}
	out := make([]migration, 0, len(byVersion))
	for _, m := range byVersion {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int64]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func cmdUp(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	pending := 0
	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if strings.TrimSpace(m.up) == "" {
			return fmt.Errorf("migration %d_%s: up.sql is empty", m.version, m.name)
		}
		fmt.Printf("applying %d_%s…\n", m.version, m.name)
		err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.up); err != nil {
				return fmt.Errorf("exec %d_%s: %w", m.version, m.name, err)
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
				m.version, m.name,
			); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		pending++
	}
	if pending == 0 {
		fmt.Println("no pending migrations")
	} else {
		fmt.Printf("applied %d migration(s)\n", pending)
	}
	return nil
}

func cmdDown(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	var latest *int64
	if err := pool.QueryRow(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&latest); err != nil {
		return err
	}
	if latest == nil {
		fmt.Println("nothing to roll back")
		return nil
	}
	for _, m := range migrations {
		if m.version != *latest {
			continue
		}
		if strings.TrimSpace(m.down) == "" {
			return fmt.Errorf("migration %d_%s has no down.sql", m.version, m.name)
		}
		fmt.Printf("rolling back %d_%s…\n", m.version, m.name)
		return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.down); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, m.version)
			return err
		})
	}
	return errors.New("latest applied version not found on disk")
}

func cmdStatus(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	fmt.Printf("%-6s %-8s %s\n", "ver", "state", "name")
	for _, m := range migrations {
		state := "pending"
		if applied[m.version] {
			state = "applied"
		}
		fmt.Printf("%-6d %-8s %s\n", m.version, state, m.name)
	}
	return nil
}

// cmdSeed populates a minimal demo dataset: one user, personal workspace,
// one client, one project, a workspace-default rate, and several historical
// time entries so the reports page is non-empty.
//
// Safe to re-run: if the demo user exists, it is reused.
func cmdSeed(ctx context.Context, pool *pgxpool.Pool) error {
	const demoEmail = "demo@timetrak.local"
	// Fixed IDs make the seed idempotent.
	userID := uuid.MustParse("00000000-0000-0000-0000-00000000a001")
	workspaceID := uuid.MustParse("00000000-0000-0000-0000-00000000b001")
	clientID := uuid.MustParse("00000000-0000-0000-0000-00000000c001")
	projectID := uuid.MustParse("00000000-0000-0000-0000-00000000d001")
	rateID := uuid.MustParse("00000000-0000-0000-0000-00000000e001")

	// Password used below; hashed fresh each seed so we don't pin a weak hash into git.
	const demoPassword = "demo-demo-demo"
	demoHash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return fmt.Errorf("hash demo password: %w", err)
	}

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, display_name)
			VALUES ($1, $2, $3, 'Demo User')
			ON CONFLICT (id) DO UPDATE SET password_hash = EXCLUDED.password_hash
		`, userID, demoEmail, demoHash)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO workspaces (id, name, slug)
			VALUES ($1, 'Demo Workspace', 'demo')
			ON CONFLICT (slug) DO NOTHING
		`, workspaceID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, 'owner')
			ON CONFLICT DO NOTHING
		`, workspaceID, userID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO clients (id, workspace_id, name, contact_email)
			VALUES ($1, $2, 'Acme Co.', 'billing@acme.example')
			ON CONFLICT (id) DO NOTHING
		`, clientID, workspaceID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO projects (id, workspace_id, client_id, name, code, default_billable)
			VALUES ($1, $2, $3, 'Website Redesign', 'WEB', true)
			ON CONFLICT (id) DO NOTHING
		`, projectID, workspaceID, clientID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO rate_rules (id, workspace_id, currency_code, hourly_rate_minor, effective_from)
			VALUES ($1, $2, 'USD', 12500, CURRENT_DATE - INTERVAL '60 days')
			ON CONFLICT (id) DO NOTHING
		`, rateID, workspaceID)
		if err != nil {
			return err
		}
		// Five historical entries over the past 10 days, each 1.5h, billable.
		now := time.Now().UTC()
		for i := 0; i < 5; i++ {
			start := now.Add(-time.Duration(i*2) * 24 * time.Hour).Add(-2 * time.Hour)
			end := start.Add(90 * time.Minute)
			_, err := tx.Exec(ctx, `
				INSERT INTO time_entries (id, workspace_id, user_id, project_id, description, started_at, ended_at, duration_seconds, is_billable)
				VALUES ($1, $2, $3, $4, 'Design review', $5, $6, $7, true)
				ON CONFLICT (id) DO NOTHING
			`, uuid.New(), workspaceID, userID, projectID, start, end, int64(end.Sub(start).Seconds()))
			if err != nil {
				return err
			}
		}
		fmt.Printf("seed complete: %s / password is %q\n", demoEmail, demoPassword)
		return nil
	})
}
