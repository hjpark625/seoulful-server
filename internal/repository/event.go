package repository

import (
	"encoding/json"
	"fmt"
	"strings"

	postgrest "github.com/supabase-community/postgrest-go"
	supabase "github.com/supabase-community/supabase-go"

	"seoulful-server-go/internal/model"
)

type EventRepository struct {
	client *supabase.Client
}

func NewEventRepository(client *supabase.Client) *EventRepository {
	return &EventRepository{client: client}
}

type EventFilter struct {
	CategorySeqs []string
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

func (r *EventRepository) FindEvents(filter EventFilter) ([]model.EventRow, int64, error) {
	query := r.client.From("events").Select("*", "exact", false)

	if len(filter.CategorySeqs) > 0 {
		query = query.In("category_seq", filter.CategorySeqs)
	}

	if filter.Search != "" {
		orFilter := fmt.Sprintf("event_name.ilike.%%%s%%,org_name.ilike.%%%s%%", filter.Search, filter.Search)
		query = query.Or(orFilter, "")
	}

	if filter.GuSeq != "" {
		query = query.Eq("gu_seq", filter.GuSeq)
	}

	if len(filter.Geohashes) > 0 {
		parts := make([]string, 0, len(filter.Geohashes))
		for _, hash := range filter.Geohashes {
			parts = append(parts, fmt.Sprintf("geohash.like.%s%%", hash))
		}
		query = query.Or(strings.Join(parts, ","), "")
	}

	if filter.WeekendStart != "" && filter.WeekendEnd != "" {
		query = query.Lte("start_date", filter.WeekendEnd)
		query = query.Gte("end_date", filter.WeekendStart)
	}

	if filter.StartDate != "" {
		query = query.Gte("end_date", filter.StartDate)
	}

	if filter.EndDate != "" {
		query = query.Lte("start_date", filter.EndDate)
	}

	query = query.Order("start_date", &postgrest.OrderOpts{Ascending: false})
	query = query.Order("event_id", &postgrest.OrderOpts{Ascending: false})

	offset := (filter.Page - 1) * filter.Limit
	query = query.Range(offset, offset+filter.Limit-1, "")

	data, count, err := query.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("supabase query failed: %w", err)
	}

	var events []model.EventRow
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return events, count, nil
}

func (r *EventRepository) FindEventByID(eventID string) (*model.EventRow, error) {
	query := r.client.From("events").Select("*", "", false).Eq("event_id", eventID)

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("supabase query failed: %w", err)
	}

	var events []model.EventRow
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if len(events) == 0 {
		return nil, nil
	}

	return &events[0], nil
}
