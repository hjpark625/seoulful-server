package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"seoulful-server-go/internal/model"
)

const eventColumns = `
	event_id, category_seq, gu_seq, event_name, period, place, org_name,
	use_target, ticket_price, inqury_number AS inquiry_number, player, describe, etc_desc,
	homepage_link, main_img, reg_date, is_public, start_date, end_date,
	theme, latitude, longitude, is_free, detail_url, geohash, display_time`

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

type EventFilter struct {
	CategorySeqs []int
	Search       string
	GuSeq        string
	Geohashes    []string
	WeekendStart string
	WeekendEnd   string
	StartDate    string
	EndDate      string
	Page         int
	Limit        int
}

func (r *EventRepository) FindEvents(ctx context.Context, filter EventFilter) ([]model.EventRow, int64, error) {
	whereClause, args := buildFilter(filter)

	var count int64
	countQuery := `SELECT COUNT(*) FROM events` + whereClause
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&count); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}

	offset := (filter.Page - 1) * filter.Limit
	pageArgs := append(args, filter.Limit, offset)
	query := `SELECT ` + eventColumns + ` FROM events` + whereClause +
		fmt.Sprintf(" ORDER BY start_date DESC NULLS LAST, event_id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	rows, err := r.pool.Query(ctx, query, pageArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]model.EventRow, 0)
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate events: %w", err)
	}

	return events, count, nil
}

func (r *EventRepository) FindEventByID(ctx context.Context, eventID int) (*model.EventRow, error) {
	query := `SELECT ` + eventColumns + ` FROM events WHERE event_id = $1`
	event, err := scanEvent(r.pool.QueryRow(ctx, query, eventID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return event, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row rowScanner) (*model.EventRow, error) {
	var event model.EventRow
	if err := row.Scan(
		&event.EventID,
		&event.CategorySeq,
		&event.GuSeq,
		&event.EventName,
		&event.Period,
		&event.Place,
		&event.OrgName,
		&event.UseTarget,
		&event.TicketPrice,
		&event.InquiryNumber,
		&event.Player,
		&event.Describe,
		&event.EtcDesc,
		&event.HomepageLink,
		&event.MainImg,
		&event.RegDate,
		&event.IsPublic,
		&event.StartDate,
		&event.EndDate,
		&event.Theme,
		&event.Latitude,
		&event.Longitude,
		&event.IsFree,
		&event.DetailURL,
		&event.Geohash,
		&event.DisplayTime,
	); err != nil {
		return nil, fmt.Errorf("scan event: %w", err)
	}

	return &event, nil
}

func buildFilter(filter EventFilter) (string, []any) {
	conditions := make([]string, 0)
	args := make([]any, 0)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if len(filter.CategorySeqs) > 0 {
		conditions = append(conditions, "category_seq = ANY("+addArg(filter.CategorySeqs)+"::integer[])")
	}
	if filter.Search != "" {
		placeholder := addArg("%" + filter.Search + "%")
		conditions = append(conditions, "(event_name ILIKE "+placeholder+" OR COALESCE(org_name, '') ILIKE "+placeholder+")")
	}
	if filter.GuSeq != "" {
		conditions = append(conditions, "gu_seq = "+addArg(filter.GuSeq)+"::integer")
	}
	if len(filter.Geohashes) > 0 {
		prefixes := make([]string, 0, len(filter.Geohashes))
		for _, geohash := range filter.Geohashes {
			prefixes = append(prefixes, geohash+"%")
		}
		conditions = append(conditions, "geohash LIKE ANY("+addArg(prefixes)+"::text[])")
	}
	if filter.WeekendStart != "" && filter.WeekendEnd != "" {
		conditions = append(conditions, "start_date <= "+addArg(filter.WeekendEnd)+"::timestamptz")
		conditions = append(conditions, "end_date >= "+addArg(filter.WeekendStart)+"::timestamptz")
	}
	if filter.StartDate != "" {
		conditions = append(conditions, "end_date >= "+addArg(filter.StartDate)+"::timestamptz")
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "start_date <= "+addArg(filter.EndDate)+"::timestamptz")
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}
