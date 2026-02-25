package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoadboardStore struct {
	pool *pgxpool.Pool
}

func NewLoadboardStore(pool *pgxpool.Pool) *LoadboardStore {
	return &LoadboardStore{pool: pool}
}

const listingColumns = `id, poster_company_id, poster_user_id, source_order_id, listing_number, title,
	origin_name, origin_city, origin_state, origin_zip,
	dest_name, dest_city, dest_state, dest_zip,
	carrier_pay, pickup_date_from, pickup_date_to, deliver_date_from, deliver_date_to,
	vehicle_count, equipment_type, special_instructions, auto_accept, status, expires_at,
	poster_company_name, poster_scac, poster_mc_number,
	created_at, updated_at`

func scanListing(row interface{ Scan(dest ...any) error }) (*models.LoadboardListing, error) {
	var l models.LoadboardListing
	err := row.Scan(
		&l.ID, &l.PosterCompanyID, &l.PosterUserID, &l.SourceOrderID, &l.ListingNumber, &l.Title,
		&l.OriginName, &l.OriginCity, &l.OriginState, &l.OriginZip,
		&l.DestName, &l.DestCity, &l.DestState, &l.DestZip,
		&l.CarrierPay, &l.PickupDateFrom, &l.PickupDateTo, &l.DeliverDateFrom, &l.DeliverDateTo,
		&l.VehicleCount, &l.EquipmentType, &l.SpecialInstructions, &l.AutoAccept, &l.Status, &l.ExpiresAt,
		&l.PosterCompanyName, &l.PosterSCAC, &l.PosterMCNumber,
		&l.CreatedAt, &l.UpdatedAt,
	)
	return &l, err
}

// ListAvailable returns posted listings visible to all companies, excluding the caller's own.
// This is the only cross-company query in the system.
func (s *LoadboardStore) ListAvailable(ctx context.Context, f models.LoadboardFilter, excludeCompanyID int) (*models.LoadboardListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	qb := newQueryBuilder()
	qb.AddRaw("status = 'Posted'")
	qb.Add("poster_company_id != ?", excludeCompanyID)
	qb.AddRaw("(expires_at IS NULL OR expires_at > NOW())")

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(title ILIKE ? OR listing_number ILIKE ? OR origin_city ILIKE ? OR dest_city ILIKE ? OR poster_company_name ILIKE ?)",
			search, search, search, search, search)
	}
	if f.OriginState != "" {
		qb.Add("origin_state = ?", f.OriginState)
	}
	if f.DestState != "" {
		qb.Add("dest_state = ?", f.DestState)
	}
	if f.MinPay != "" {
		qb.Add("carrier_pay >= ?::numeric", f.MinPay)
	}
	if f.MaxPay != "" {
		qb.Add("carrier_pay <= ?::numeric", f.MaxPay)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM loadboard_listings "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count listings: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM loadboard_listings %s ORDER BY created_at DESC %s",
		listingColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list available listings: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardListing, error) {
		l, err := scanListing(row)
		if err != nil {
			return models.LoadboardListing{}, err
		}
		return *l, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan listing: %w", err)
	}

	return &models.LoadboardListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

// GetByID returns a single listing (cross-company, any authenticated user).
func (s *LoadboardStore) GetByID(ctx context.Context, id int) (*models.LoadboardListing, error) {
	query := fmt.Sprintf("SELECT %s FROM loadboard_listings WHERE id = $1", listingColumns)
	l, err := scanListing(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get listing %d: %w", id, err)
	}
	return l, nil
}

// GetByIDForUpdate returns a listing with a row lock for claim operations.
func (s *LoadboardStore) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id int) (*models.LoadboardListing, error) {
	query := fmt.Sprintf("SELECT %s FROM loadboard_listings WHERE id = $1 FOR UPDATE", listingColumns)
	l, err := scanListing(tx.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get listing for update %d: %w", id, err)
	}
	return l, nil
}

// GetListingVehicles returns vehicles for a listing (cross-company).
func (s *LoadboardStore) GetListingVehicles(ctx context.Context, listingID int) ([]models.LoadboardListingVehicle, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, listing_id, source_vehicle_id, vin, year, make, model, color, weight, category, body_style, operable, run_drive, created_at
		FROM loadboard_listing_vehicles WHERE listing_id = $1 ORDER BY id`, listingID)
	if err != nil {
		return nil, fmt.Errorf("get listing vehicles %d: %w", listingID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardListingVehicle, error) {
		var v models.LoadboardListingVehicle
		if err := row.Scan(&v.ID, &v.ListingID, &v.SourceVehicleID, &v.VIN, &v.Year, &v.Make, &v.Model,
			&v.Color, &v.Weight, &v.Category, &v.BodyStyle, &v.Operable, &v.RunDrive, &v.CreatedAt); err != nil {
			return models.LoadboardListingVehicle{}, err
		}
		return v, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan listing vehicle: %w", err)
	}
	return items, nil
}

// ListMyListings returns listings posted by the given company.
func (s *LoadboardStore) ListMyListings(ctx context.Context, companyID int, f models.LoadboardFilter) (*models.LoadboardListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	qb := newQueryBuilder()
	qb.Add("poster_company_id = ?", companyID)

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(title ILIKE ? OR listing_number ILIKE ?)", search, search)
	}
	if f.Status != "" {
		qb.Add("status = ?", f.Status)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM loadboard_listings "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count my listings: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM loadboard_listings %s ORDER BY created_at DESC %s",
		listingColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list my listings: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardListing, error) {
		l, err := scanListing(row)
		if err != nil {
			return models.LoadboardListing{}, err
		}
		return *l, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan my listing: %w", err)
	}

	return &models.LoadboardListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

// ListMyClaims returns claims made by the given carrier company.
func (s *LoadboardStore) ListMyClaims(ctx context.Context, companyID int, f models.LoadboardFilter) (*models.LoadboardClaimListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	qb := newQueryBuilder()
	qb.Add("c.carrier_company_id = ?", companyID)

	if f.Search != "" {
		search := "%" + f.Search + "%"
		qb.Add("(l.listing_number ILIKE ? OR l.title ILIKE ?)", search, search)
	}
	if f.Status != "" {
		qb.Add("c.status = ?", f.Status)
	}

	var total int
	if err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM loadboard_claims c JOIN loadboard_listings l ON l.id = c.listing_id "+qb.Where(),
		qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count my claims: %w", err)
	}

	query := fmt.Sprintf(`SELECT %s, l.listing_number, l.title, l.status,
			COALESCE(uc.cnt, 0) AS unread_count
		FROM loadboard_claims c
		JOIN loadboard_listings l ON l.id = c.listing_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM loadboard_messages m
			WHERE m.claim_id = c.id
			  AND m.sender_company_id = l.poster_company_id
			  AND (c.carrier_last_read_at IS NULL OR m.created_at > c.carrier_last_read_at)
		) uc ON true
		%s ORDER BY c.created_at DESC %s`,
		claimColumnsAliased(), qb.Where(), qb.Paginate(f.PageSize, f.Page))

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list my claims: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardClaim, error) {
		var c models.LoadboardClaim
		if err := row.Scan(
			&c.ID, &c.ListingID, &c.CarrierCompanyID, &c.CarrierUserID,
			&c.CarrierCompanyName, &c.CarrierSCAC, &c.CarrierMCNumber, &c.CarrierDOTNumber, &c.CarrierInsuranceExp,
			&c.CarrierOrderID, &c.AgreedPay, &c.VehicleCount, &c.Status,
			&c.CarrierNotes, &c.PosterNotes,
			&c.AcceptedAt, &c.RejectedAt, &c.CancelledAt, &c.CompletedAt,
			&c.CreatedAt, &c.UpdatedAt,
			&c.ListingNumber, &c.ListingTitle, &c.ListingStatus,
			&c.UnreadCount,
		); err != nil {
			return models.LoadboardClaim{}, err
		}
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan my claim: %w", err)
	}

	return &models.LoadboardClaimListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

// ListClaimsOnListing returns all claims on a specific listing, including message and unread counts.
func (s *LoadboardStore) ListClaimsOnListing(ctx context.Context, listingID int) ([]models.LoadboardClaim, error) {
	query := fmt.Sprintf(`SELECT %s,
			COALESCE(mc.cnt, 0) AS message_count,
			COALESCE(uc.cnt, 0) AS unread_count
		FROM loadboard_claims c
		LEFT JOIN (SELECT claim_id, COUNT(*) AS cnt FROM loadboard_messages GROUP BY claim_id) mc ON mc.claim_id = c.id
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM loadboard_messages m
			WHERE m.claim_id = c.id
			  AND m.sender_company_id = c.carrier_company_id
			  AND (c.poster_last_read_at IS NULL OR m.created_at > c.poster_last_read_at)
		) uc ON true
		WHERE c.listing_id = $1 ORDER BY c.created_at DESC`,
		claimColumnsAliased())
	rows, err := s.pool.Query(ctx, query, listingID)
	if err != nil {
		return nil, fmt.Errorf("list claims on listing %d: %w", listingID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardClaim, error) {
		var c models.LoadboardClaim
		if err := row.Scan(
			&c.ID, &c.ListingID, &c.CarrierCompanyID, &c.CarrierUserID,
			&c.CarrierCompanyName, &c.CarrierSCAC, &c.CarrierMCNumber, &c.CarrierDOTNumber, &c.CarrierInsuranceExp,
			&c.CarrierOrderID, &c.AgreedPay, &c.VehicleCount, &c.Status,
			&c.CarrierNotes, &c.PosterNotes,
			&c.AcceptedAt, &c.RejectedAt, &c.CancelledAt, &c.CompletedAt,
			&c.CreatedAt, &c.UpdatedAt,
			&c.MessageCount, &c.UnreadCount,
		); err != nil {
			return models.LoadboardClaim{}, err
		}
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan claim: %w", err)
	}
	return items, nil
}

// GetClaimByID returns a single claim.
func (s *LoadboardStore) GetClaimByID(ctx context.Context, id int) (*models.LoadboardClaim, error) {
	query := fmt.Sprintf(`SELECT %s, l.listing_number, l.title, l.status
		FROM loadboard_claims c
		JOIN loadboard_listings l ON l.id = c.listing_id
		WHERE c.id = $1`,
		claimColumnsAliased())
	var c models.LoadboardClaim
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.ListingID, &c.CarrierCompanyID, &c.CarrierUserID,
		&c.CarrierCompanyName, &c.CarrierSCAC, &c.CarrierMCNumber, &c.CarrierDOTNumber, &c.CarrierInsuranceExp,
		&c.CarrierOrderID, &c.AgreedPay, &c.VehicleCount, &c.Status,
		&c.CarrierNotes, &c.PosterNotes,
		&c.AcceptedAt, &c.RejectedAt, &c.CancelledAt, &c.CompletedAt,
		&c.CreatedAt, &c.UpdatedAt,
		&c.ListingNumber, &c.ListingTitle, &c.ListingStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("get claim %d: %w", id, err)
	}
	return &c, nil
}

// CreateListing inserts a new listing.
func (s *LoadboardStore) CreateListing(ctx context.Context, l *models.LoadboardListing) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO loadboard_listings (
			poster_company_id, poster_user_id, source_order_id, listing_number, title,
			origin_name, origin_city, origin_state, origin_zip,
			dest_name, dest_city, dest_state, dest_zip,
			carrier_pay, pickup_date_from, pickup_date_to, deliver_date_from, deliver_date_to,
			vehicle_count, equipment_type, special_instructions, auto_accept, status, expires_at,
			poster_company_name, poster_scac, poster_mc_number
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27
		) RETURNING id, created_at, updated_at`,
		l.PosterCompanyID, l.PosterUserID, l.SourceOrderID, l.ListingNumber, l.Title,
		l.OriginName, l.OriginCity, l.OriginState, l.OriginZip,
		l.DestName, l.DestCity, l.DestState, l.DestZip,
		l.CarrierPay, l.PickupDateFrom, l.PickupDateTo, l.DeliverDateFrom, l.DeliverDateTo,
		l.VehicleCount, l.EquipmentType, l.SpecialInstructions, l.AutoAccept, l.Status, l.ExpiresAt,
		l.PosterCompanyName, l.PosterSCAC, l.PosterMCNumber,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}
	return nil
}

// CreateListingVehicles bulk-inserts listing vehicles.
func (s *LoadboardStore) CreateListingVehicles(ctx context.Context, vehicles []models.LoadboardListingVehicle) error {
	for i := range vehicles {
		v := &vehicles[i]
		err := s.pool.QueryRow(ctx,
			`INSERT INTO loadboard_listing_vehicles (
				listing_id, source_vehicle_id, vin, year, make, model, color, weight, category, body_style, operable, run_drive
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id, created_at`,
			v.ListingID, v.SourceVehicleID, v.VIN, v.Year, v.Make, v.Model,
			v.Color, v.Weight, v.Category, v.BodyStyle, v.Operable, v.RunDrive,
		).Scan(&v.ID, &v.CreatedAt)
		if err != nil {
			return fmt.Errorf("create listing vehicle: %w", err)
		}
	}
	return nil
}

// CreateClaim inserts a new claim within a transaction.
func (s *LoadboardStore) CreateClaim(ctx context.Context, tx pgx.Tx, c *models.LoadboardClaim) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO loadboard_claims (
			listing_id, carrier_company_id, carrier_user_id,
			carrier_company_name, carrier_scac, carrier_mc_number, carrier_dot_number, carrier_insurance_exp,
			carrier_order_id, agreed_pay, vehicle_count, status,
			carrier_notes, accepted_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, created_at, updated_at`,
		c.ListingID, c.CarrierCompanyID, c.CarrierUserID,
		c.CarrierCompanyName, c.CarrierSCAC, c.CarrierMCNumber, c.CarrierDOTNumber, c.CarrierInsuranceExp,
		c.CarrierOrderID, c.AgreedPay, c.VehicleCount, c.Status,
		c.CarrierNotes, c.AcceptedAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create claim: %w", err)
	}
	return nil
}

// UpdateListingStatus updates a listing's status.
func (s *LoadboardStore) UpdateListingStatus(ctx context.Context, id int, status string) error {
	result, err := s.pool.Exec(ctx,
		"UPDATE loadboard_listings SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("update listing status %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("listing %d not found", id)
	}
	return nil
}

// UpdateListingStatusTx updates a listing's status within a transaction.
func (s *LoadboardStore) UpdateListingStatusTx(ctx context.Context, tx pgx.Tx, id int, status string) error {
	result, err := tx.Exec(ctx,
		"UPDATE loadboard_listings SET status = $1, updated_at = NOW() WHERE id = $2", status, id)
	if err != nil {
		return fmt.Errorf("update listing status %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("listing %d not found", id)
	}
	return nil
}

// UpdateClaimStatus updates a claim's status and the corresponding timestamp.
func claimStatusDateCol(status string) (string, error) {
	switch status {
	case "Accepted":
		return "accepted_at", nil
	case "Rejected":
		return "rejected_at", nil
	case "Cancelled":
		return "cancelled_at", nil
	case "Completed":
		return "completed_at", nil
	default:
		return "", fmt.Errorf("unrecognized claim status: %q", status)
	}
}

func (s *LoadboardStore) UpdateClaimStatus(ctx context.Context, id int, status string) error {
	dateCol, err := claimStatusDateCol(status)
	if err != nil {
		return err
	}
	now := time.Now()
	query := fmt.Sprintf("UPDATE loadboard_claims SET status = $1, %s = $2, updated_at = NOW() WHERE id = $3", dateCol)
	result, err := s.pool.Exec(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("update claim status %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim %d not found", id)
	}
	return nil
}

// UpdateClaimStatusTx updates a claim's status within a transaction.
func (s *LoadboardStore) UpdateClaimStatusTx(ctx context.Context, tx pgx.Tx, id int, status string) error {
	dateCol, err := claimStatusDateCol(status)
	if err != nil {
		return err
	}
	now := time.Now()
	query := fmt.Sprintf("UPDATE loadboard_claims SET status = $1, %s = $2, updated_at = NOW() WHERE id = $3", dateCol)
	result, err := tx.Exec(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("update claim status %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim %d not found", id)
	}
	return nil
}

// UpdateClaimCarrierOrder sets the carrier_order_id on a claim within a transaction.
func (s *LoadboardStore) UpdateClaimCarrierOrder(ctx context.Context, tx pgx.Tx, claimID int, orderID int) error {
	_, err := tx.Exec(ctx,
		"UPDATE loadboard_claims SET carrier_order_id = $1, updated_at = NOW() WHERE id = $2",
		orderID, claimID)
	if err != nil {
		return fmt.Errorf("update claim carrier order: %w", err)
	}
	return nil
}

// NextListingNumber generates the next listing number from the global sequence.
func (s *LoadboardStore) NextListingNumber(ctx context.Context) (string, error) {
	var seq int
	err := s.pool.QueryRow(ctx, "SELECT nextval('loadboard_listing_number_seq')").Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("next listing number: %w", err)
	}
	return fmt.Sprintf("LB-%06d", seq), nil
}

// ExpireListings finds posted listings past their expiration and marks them expired.
// Returns the number of listings expired.
func (s *LoadboardStore) ExpireListings(ctx context.Context) (int, error) {
	// Get IDs of expired listings
	rows, err := s.pool.Query(ctx,
		"SELECT id FROM loadboard_listings WHERE status = 'Posted' AND expires_at IS NOT NULL AND expires_at < NOW()")
	if err != nil {
		return 0, fmt.Errorf("find expired listings: %w", err)
	}
	ids, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int, error) {
		var id int
		return id, row.Scan(&id)
	})
	if err != nil {
		return 0, fmt.Errorf("scan expired listing ids: %w", err)
	}

	for _, id := range ids {
		// Reject any pending claims
		if _, err := s.pool.Exec(ctx,
			"UPDATE loadboard_claims SET status = 'Rejected', rejected_at = NOW(), poster_notes = 'Listing expired', updated_at = NOW() WHERE listing_id = $1 AND status = 'Pending'",
			id); err != nil {
			return 0, fmt.Errorf("reject pending claims for expired listing %d: %w", id, err)
		}
		if err := s.UpdateListingStatus(ctx, id, "Expired"); err != nil {
			return 0, fmt.Errorf("expire listing %d: %w", id, err)
		}
	}

	return len(ids), nil
}

// ListMessagesByClaim returns all messages for a claim, ordered chronologically.
func (s *LoadboardStore) ListMessagesByClaim(ctx context.Context, claimID int) ([]models.LoadboardMessage, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, claim_id, sender_company_id, sender_user_id, sender_name, body, created_at
		FROM loadboard_messages WHERE claim_id = $1 ORDER BY created_at ASC`, claimID)
	if err != nil {
		return nil, fmt.Errorf("list messages for claim %d: %w", claimID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardMessage, error) {
		var m models.LoadboardMessage
		if err := row.Scan(&m.ID, &m.ClaimID, &m.SenderCompanyID, &m.SenderUserID, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
			return models.LoadboardMessage{}, err
		}
		return m, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan message: %w", err)
	}
	return items, nil
}

// CreateMessage inserts a new message on a claim.
func (s *LoadboardStore) CreateMessage(ctx context.Context, m *models.LoadboardMessage) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO loadboard_messages (claim_id, sender_company_id, sender_user_id, sender_name, body)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		m.ClaimID, m.SenderCompanyID, m.SenderUserID, m.SenderName, m.Body,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	return nil
}

// UpdateClaimLastRead sets the poster_last_read_at or carrier_last_read_at to NOW().
func (s *LoadboardStore) UpdateClaimLastRead(ctx context.Context, claimID int, isPoster bool) error {
	col := "carrier_last_read_at"
	if isPoster {
		col = "poster_last_read_at"
	}
	query := fmt.Sprintf("UPDATE loadboard_claims SET %s = NOW() WHERE id = $1", col)
	_, err := s.pool.Exec(ctx, query, claimID)
	if err != nil {
		return fmt.Errorf("update claim last read: %w", err)
	}
	return nil
}

// CountUnreadMessages returns the number of unread loadboard messages for a company.
// A message is "unread" if it was sent by the other party and created after my last_read_at.
func (s *LoadboardStore) CountUnreadMessages(ctx context.Context, companyID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM loadboard_messages m
		JOIN loadboard_claims c ON c.id = m.claim_id
		JOIN loadboard_listings l ON l.id = c.listing_id
		WHERE m.sender_company_id != $1
		  AND (
		    (l.poster_company_id = $1 AND (c.poster_last_read_at IS NULL OR m.created_at > c.poster_last_read_at))
		    OR
		    (c.carrier_company_id = $1 AND (c.carrier_last_read_at IS NULL OR m.created_at > c.carrier_last_read_at))
		  )`
	var count int
	if err := s.pool.QueryRow(ctx, query, companyID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread messages: %w", err)
	}
	return count, nil
}

// claimColumnsAliased returns claim column names with c. prefix for JOIN queries.
func claimColumnsAliased() string {
	return `c.id, c.listing_id, c.carrier_company_id, c.carrier_user_id,
	c.carrier_company_name, c.carrier_scac, c.carrier_mc_number, c.carrier_dot_number, c.carrier_insurance_exp,
	c.carrier_order_id, c.agreed_pay, c.vehicle_count, c.status,
	c.carrier_notes, c.poster_notes,
	c.accepted_at, c.rejected_at, c.cancelled_at, c.completed_at,
	c.created_at, c.updated_at`
}
