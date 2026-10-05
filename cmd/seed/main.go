package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chuuch/gorest/pkg/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	defaultOrgName  = "Zyntera"
	defaultPassword = "password12"
	bcryptCost      = 10
)

func main() {
	_ = godotenv.Load()

	orgName := flag.String("org", defaultOrgName, "organization name to seed")
	passwordPlain := flag.String("password", defaultPassword, "password for seeded members/portal users")
	force := flag.Bool("force", false, "re-seed even if Northwind client already exists")
	flag.Parse()

	dbURL := os.Getenv("GOREST_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("GOREST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	var orgID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM organizations WHERE name = $1`, *orgName).Scan(&orgID)
	if err != nil {
		log.Fatalf("organization %q not found: %v", *orgName, err)
	}

	var ownerID uuid.UUID
	err = pool.QueryRow(ctx, `
		SELECT u.id
		FROM memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1 AND m.role = 'owner'
		LIMIT 1
	`, orgID).Scan(&ownerID)
	if err != nil {
		log.Fatalf("owner for %q not found: %v", *orgName, err)
	}

	var existing int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM clients
		WHERE organization_id = $1 AND name = 'Northwind'
	`, orgID).Scan(&existing)
	if existing > 0 && !*force {
		log.Printf("%q already has client Northwind — pass -force to re-seed", *orgName)
		os.Exit(0)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	if *force {
		if err := wipeSeedClients(ctx, tx, orgID); err != nil {
			log.Fatalf("wipe: %v", err)
		}
	}

	hasher := password.NewBcryptHasher(bcryptCost)
	hash, err := hasher.Hash(*passwordPlain)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	now := time.Now().UTC()

	adminID, err := upsertStaff(ctx, tx, "admin@zyntera.seed", "Alex Admin", hash, orgID, "admin", now)
	if err != nil {
		log.Fatalf("admin: %v", err)
	}
	memberID, err := upsertStaff(ctx, tx, "member@zyntera.seed", "Morgan Member", hash, orgID, "member", now)
	if err != nil {
		log.Fatalf("member: %v", err)
	}

	northwindID := uuid.New()
	contosoID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO clients (
			id, organization_id, name, notes,
			legal_name, vat_id, address_line1, address_line2, city, postal_code, country,
			created_at, updated_at
		) VALUES
		($1, $2, 'Northwind', 'Primary retainer client',
		 'Northwind Traders OÜ', 'EE123456789', 'Narva mnt 5', '', 'Tallinn', '10117', 'EE',
		 $3, $3),
		($4, $2, 'Contoso', 'Project-based client',
		 'Contoso Ltd', 'GB987654321', '1 Contoso Way', '', 'London', 'EC1A 1BB', 'GB',
		 $3, $3)
	`, northwindID, orgID, now, contosoID)
	if err != nil {
		log.Fatalf("clients: %v", err)
	}

	portalID, err := upsertPortal(ctx, tx, "portal@northwind.seed", "Casey Client", hash, orgID, northwindID, now)
	if err != nil {
		log.Fatalf("portal user: %v", err)
	}

	websiteID := uuid.New()
	brandID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO projects (id, organization_id, client_id, name, notes, created_at, updated_at)
		VALUES
		($1, $2, $3, 'Website Redesign', 'Marketing site refresh', $5, $5),
		($4, $2, $3, 'Brand Kit', 'Logo and guidelines', $5, $5)
	`, websiteID, orgID, northwindID, brandID, now)
	if err != nil {
		log.Fatalf("projects: %v", err)
	}

	ticketOpenID := uuid.New()
	ticketProgressID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO tickets (
			id, organization_id, client_id, user_id, kind, status, title, body, created_at, updated_at
		) VALUES
		($1, $2, $3, $4, 'bug', 'open', 'Checkout button broken', 'Mobile Safari cannot complete checkout.', $6, $6),
		($5, $2, $3, $4, 'feature', 'in_progress', 'Add invoice PDF export', 'Clients want downloadable PDFs from the portal.', $6, $6)
	`, ticketOpenID, orgID, northwindID, portalID, ticketProgressID, now)
	if err != nil {
		log.Fatalf("tickets: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ticket_comments (
			id, organization_id, ticket_id, user_id, body, created_at, updated_at
		) VALUES
		($1, $2, $3, $4, 'Looking into the Safari checkout issue now.', $6, $6),
		($5, $2, $3, $7, 'Thanks — happens on iPhone 15.', $6, $6)
	`, uuid.New(), orgID, ticketOpenID, adminID, uuid.New(), now, portalID)
	if err != nil {
		log.Fatalf("ticket comments: %v", err)
	}

	taskTodoID := uuid.New()
	taskProgressID := uuid.New()
	taskDoneID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO tasks (
			id, organization_id, project_id, title, notes, status,
			created_at, updated_at, completed_at, ticket_id, version, created_by, assignee_id
		) VALUES
		($1, $2, $3, 'Fix mobile checkout', 'Reproduce on Safari', 'todo',
		 $8, $8, NULL, $4, 1, $5, $6),
		($9, $2, $3, 'Wire invoice PDF', 'Portal download button', 'in_progress',
		 $8, $8, NULL, $7, 1, $5, $5),
		($10, $2, $11, 'Deliver logo pack', 'SVG + PNG exports', 'done',
		 $8, $8, $8, NULL, 1, $5, $6)
	`, taskTodoID, orgID, websiteID, ticketOpenID, ownerID, memberID, ticketProgressID, now,
		taskProgressID, taskDoneID, brandID)
	if err != nil {
		log.Fatalf("tasks: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO comments (id, organization_id, task_id, user_id, body, created_at, updated_at)
		VALUES
		($1, $2, $3, $4, 'Reproduced on iOS 18 — CSS overflow on the CTA.', $6, $6),
		($5, $2, $3, $7, 'Patch staged on preview.', $6, $6)
	`, uuid.New(), orgID, taskTodoID, memberID, uuid.New(), now, adminID)
	if err != nil {
		log.Fatalf("comments: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO time_entries (id, organization_id, task_id, user_id, minutes, notes, created_at, updated_at)
		VALUES
		($1, $2, $3, $4, 90, 'Safari checkout debug', $8, $8),
		($5, $2, $6, $7, 120, 'PDF endpoint scaffold', $8, $8),
		($9, $2, $10, $4, 60, 'Logo export packaging', $8, $8)
	`, uuid.New(), orgID, taskTodoID, memberID,
		uuid.New(), taskProgressID, adminID, now,
		uuid.New(), taskDoneID)
	if err != nil {
		log.Fatalf("time entries: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_events (
			id, organization_id, actor_id, action, entity_type, entity_id, summary, created_at
		) VALUES
		($1, $2, $3, 'created', 'ticket', $4, 'Casey Client opened ticket Checkout button broken', $9),
		($5, $2, $6, 'created', 'task', $7, 'Owner created task Fix mobile checkout', $9),
		($8, $2, $6, 'converted', 'ticket', $10, 'Alex Admin converted ticket to task', $9)
	`, uuid.New(), orgID, portalID, ticketOpenID,
		uuid.New(), ownerID, taskTodoID,
		uuid.New(), now, ticketProgressID)
	if err != nil {
		log.Fatalf("activity: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (
			id, organization_id, recipient_id, actor_id, kind, entity_type, entity_id, summary, read_at, created_at
		) VALUES
		($1, $2, $3, $4, 'ticket_opened', 'ticket', $5, 'New ticket: Checkout button broken', NULL, $9),
		($6, $2, $7, $3, 'task_assigned', 'task', $8, 'You were assigned Fix mobile checkout', NULL, $9),
		($10, $2, $3, $7, 'task_commented', 'comment', $11, 'Morgan Member commented on Fix mobile checkout', $9, $9)
	`, uuid.New(), orgID, ownerID, portalID, ticketOpenID,
		uuid.New(), memberID, taskTodoID, now,
		uuid.New(), uuid.New())
	if err != nil {
		log.Fatalf("notifications: %v", err)
	}

	invoiceID := uuid.New()
	periodFrom := now.AddDate(0, 0, -14)
	periodTo := now
	issued := now.AddDate(0, 0, -1)
	due := now.AddDate(0, 0, 13)
	subtotal := 13500
	_, err = tx.Exec(ctx, `
		INSERT INTO invoices (
			id, organization_id, client_id, number, status, currency, rate_cents,
			organization_name, client_name,
			seller_legal_name, buyer_legal_name, buyer_country, vat_regime, vat_rate_bps,
			subtotal_cents, vat_cents, bank_iban, bank_bic, bank_name,
			period_from, period_to, issued_at, due_at, sent_at, paid_at,
			total_minutes, total_cents, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'INV-2026-0001', 'sent', 'EUR', 3000,
			$4, 'Northwind',
			$4, 'Northwind Traders OÜ', 'EE', 'untaxed', 0,
			$5, 0, '', '', '',
			$6, $7, $8, $9, $8, NULL,
			270, $5, $10, $10
		)
	`, invoiceID, orgID, northwindID, *orgName, subtotal, periodFrom, periodTo, issued, due, now)
	if err != nil {
		log.Fatalf("invoice: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO invoice_line_items (
			id, invoice_id, organization_id, project_name, task_title, minutes, amount_cents, position, created_at
		) VALUES
		($1, $2, $3, 'Website Redesign', 'Fix mobile checkout', 90, 4500, 1, $5),
		($4, $2, $3, 'Website Redesign', 'Wire invoice PDF', 120, 6000, 2, $5),
		($6, $2, $3, 'Brand Kit', 'Deliver logo pack', 60, 3000, 3, $5)
	`, uuid.New(), invoiceID, orgID, uuid.New(), now, uuid.New())
	if err != nil {
		log.Fatalf("invoice lines: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("Seeded organization %q (%s)\n", *orgName, orgID)
	fmt.Println("Staff logins (staff app):")
	fmt.Println("  owner:  existing Zyntera owner account")
	fmt.Printf("  admin:  admin@zyntera.seed / %s\n", *passwordPlain)
	fmt.Printf("  member: member@zyntera.seed / %s\n", *passwordPlain)
	fmt.Println("Portal login (client portal):")
	fmt.Printf("  portal@northwind.seed / %s\n", *passwordPlain)
	fmt.Println("Data: Northwind + Contoso, 2 projects, 3 tasks, 2 tickets, time, activity, notifications, INV-2026-0001")
}

func wipeSeedClients(ctx context.Context, tx pgx.Tx, orgID uuid.UUID) error {
	rows, err := tx.Query(ctx, `
		SELECT id FROM clients
		WHERE organization_id = $1 AND name IN ('Northwind', 'Contoso')
	`, orgID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var clientIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		clientIDs = append(clientIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(clientIDs) == 0 {
		return nil
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM invoices
		WHERE organization_id = $1 AND client_id = ANY($2)
	`, orgID, clientIDs); err != nil {
		return fmt.Errorf("delete invoices: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM clients WHERE id = ANY($1)`, clientIDs); err != nil {
		return fmt.Errorf("delete clients: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM users WHERE email = 'portal@northwind.seed'
	`); err != nil {
		return fmt.Errorf("delete portal user: %w", err)
	}
	return nil
}

func upsertStaff(
	ctx context.Context,
	tx pgx.Tx,
	email, displayName, hash string,
	orgID uuid.UUID,
	role string,
	now time.Time,
) (uuid.UUID, error) {
	id := uuid.New()
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&existingID)
	if err == nil {
		id = existingID
		_, err = tx.Exec(ctx, `
			UPDATE users
			SET display_name = $2, password_hash = $3, updated_at = $4
			WHERE id = $1
		`, id, displayName, hash, now)
		if err != nil {
			return uuid.Nil, err
		}
	} else if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
		`, id, email, displayName, hash, now)
		if err != nil {
			return uuid.Nil, err
		}
	} else {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO memberships (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, uuid.New(), orgID, id, role, now)
	return id, err
}

func upsertPortal(
	ctx context.Context,
	tx pgx.Tx,
	email, displayName, hash string,
	orgID, clientID uuid.UUID,
	now time.Time,
) (uuid.UUID, error) {
	id := uuid.New()
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, email).Scan(&existingID)
	if err == nil {
		id = existingID
		_, err = tx.Exec(ctx, `
			UPDATE users
			SET display_name = $2, password_hash = $3, updated_at = $4
			WHERE id = $1
		`, id, displayName, hash, now)
		if err != nil {
			return uuid.Nil, err
		}
	} else if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (id, email, display_name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
		`, id, email, displayName, hash, now)
		if err != nil {
			return uuid.Nil, err
		}
	} else {
		return uuid.Nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO client_users (id, organization_id, client_id, user_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET client_id = EXCLUDED.client_id, organization_id = EXCLUDED.organization_id
	`, uuid.New(), orgID, clientID, id, now)
	return id, err
}
