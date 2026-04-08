package aipolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pool     *pgxpool.Pool
	poolOnce sync.Once
	poolErr  error
)

func dsn() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("DBUSER"),
		os.Getenv("DBPASSWORD"),
		os.Getenv("DBHOST"),
		os.Getenv("DBPORT"),
		os.Getenv("DBNAME"),
		os.Getenv("SSLMode"),
	)
}

func db() (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		pool, poolErr = pgxpool.New(context.Background(), dsn())
	})
	return pool, poolErr
}

// pgScanner is satisfied by both pgx.Row and pgx.Rows.
type pgScanner interface {
	Scan(dest ...any) error
}

func scanPolicy(row pgScanner) (*Policy, error) {
	var p Policy
	var condBytes []byte
	err := row.Scan(
		&p.ID, &p.PolicyID, &p.Name,
		&p.RemoteMCPService, &p.ResourceAccessRequest, &p.Environment,
		&p.Enabled, &p.Priority, &condBytes, &p.Description,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(condBytes, &p.Conditions); err != nil {
		return nil, fmt.Errorf("unmarshal conditions: %w", err)
	}
	return &p, nil
}

const selectCols = `id, policy_id, name, remote_mcp_service, resource_access_request,
	environment, enabled, priority, conditions, description, created_at, updated_at`

func createPolicy(p *Policy) error {
	q, err := db()
	if err != nil {
		return err
	}
	condJSON, err := json.Marshal(p.Conditions)
	if err != nil {
		return err
	}
	p.ID = uuid.New()
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err = q.Exec(context.Background(),
		`INSERT INTO policies
			(id, policy_id, name, remote_mcp_service, resource_access_request,
			 environment, enabled, priority, conditions, description, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		p.ID, p.PolicyID, p.Name, p.RemoteMCPService, p.ResourceAccessRequest,
		p.Environment, p.Enabled, p.Priority, condJSON, p.Description,
		p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func getPolicy(id uuid.UUID) (*Policy, error) {
	q, err := db()
	if err != nil {
		return nil, err
	}
	row := q.QueryRow(context.Background(),
		`SELECT `+selectCols+` FROM policies WHERE id=$1`, id)
	p, err := scanPolicy(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func updatePolicy(id uuid.UUID, p *Policy) error {
	q, err := db()
	if err != nil {
		return err
	}
	condJSON, err := json.Marshal(p.Conditions)
	if err != nil {
		return err
	}
	p.UpdatedAt = time.Now().UTC()
	tag, err := q.Exec(context.Background(),
		`UPDATE policies SET
			policy_id=$2, name=$3, remote_mcp_service=$4, resource_access_request=$5,
			environment=$6, enabled=$7, priority=$8, conditions=$9,
			description=$10, updated_at=$11
		 WHERE id=$1`,
		id, p.PolicyID, p.Name, p.RemoteMCPService, p.ResourceAccessRequest,
		p.Environment, p.Enabled, p.Priority, condJSON, p.Description, p.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func deletePolicy(id uuid.UUID) error {
	q, err := db()
	if err != nil {
		return err
	}
	tag, err := q.Exec(context.Background(),
		`DELETE FROM policies WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// findEnabledPolicies returns all enabled policies matching the three lookup
// fields, ordered by ascending priority (lower number = higher priority).
func findEnabledPolicies(service, resource, env string) ([]Policy, error) {
	q, err := db()
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(context.Background(),
		`SELECT `+selectCols+`
		 FROM policies
		 WHERE enabled=TRUE
		   AND remote_mcp_service=$1
		   AND resource_access_request=$2
		   AND environment=$3
		 ORDER BY priority ASC`,
		service, resource, env,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, *p)
	}
	return policies, rows.Err()
}
