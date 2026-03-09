package service

import (
	_ "time/tzdata"

	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"seoulful-server-go/internal/model"
	"seoulful-server-go/internal/repository"
)

type EventService struct {
	repo *repository.EventRepository
}

func NewEventService(repo *repository.EventRepository) *EventService {
	return &EventService{repo: repo}
}

type EventQueryParams struct {
	Category  string
	Search    string
	Weekend   string
	StartDate string
	EndDate   string
	GuSeq     string
	Geohashes string
	Page      string
	Limit     string
}

var categorySeqMap = map[string][]string{
	"FESTIVAL":    {"9", "10", "11", "12", "13"},
	"EXHIBITION":  {"8"},
	"PERFORMANCE": {"2", "3", "4", "5", "6", "7", "14", "15"},
	"OTHER":       {"1", "16"},
}

var seqToCategoryMap = map[int]model.EventCategory{
	9:  model.CategoryFestival,
	10: model.CategoryFestival,
	11: model.CategoryFestival,
	12: model.CategoryFestival,
	13: model.CategoryFestival,
	8:  model.CategoryExhibition,
	2:  model.CategoryPerformance,
	3:  model.CategoryPerformance,
	4:  model.CategoryPerformance,
	5:  model.CategoryPerformance,
	6:  model.CategoryPerformance,
	7:  model.CategoryPerformance,
	14: model.CategoryPerformance,
	15: model.CategoryPerformance,
	1:  model.CategoryOther,
	16: model.CategoryOther,
}

var reservedCharsRegex = regexp.MustCompile(`[(),]`)

func mapCategorySeqToCategory(seq int) model.EventCategory {
	if category, ok := seqToCategoryMap[seq]; ok {
		return category
	}

	return model.CategoryOther
}

func sanitizeNull(s *string) *string {
	if s == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*s)
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return nil
	}

	return &trimmed
}

func sanitizeSearch(search string) string {
	s := strings.TrimSpace(search)
	if utf8.RuneCountInString(s) > 100 {
		runes := []rune(s)
		s = string(runes[:100])
	}

	s = reservedCharsRegex.ReplaceAllString(s, " ")
	return s
}

func getWeekendRange() (fridayStart, sundayEnd string) {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		loc = time.FixedZone("KST", 9*60*60)
	}

	now := time.Now().In(loc)
	weekday := now.Weekday()

	var daysUntilFriday int
	switch {
	case weekday == time.Friday:
		daysUntilFriday = 0
	case weekday == time.Saturday:
		daysUntilFriday = -1
	case weekday == time.Sunday:
		daysUntilFriday = -2
	default:
		daysUntilFriday = int(time.Friday - weekday)
	}

	friday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilFriday, 0, 0, 0, 0, loc)
	sunday := time.Date(friday.Year(), friday.Month(), friday.Day()+2, 23, 59, 59, 0, loc)

	return friday.Format(time.RFC3339), sunday.Format(time.RFC3339)
}

func (s *EventService) BuildFilter(params EventQueryParams) repository.EventFilter {
	filter := repository.EventFilter{Page: 1, Limit: 20}

	if params.Category != "" {
		for _, category := range strings.Split(params.Category, ",") {
			category = strings.TrimSpace(strings.ToUpper(category))
			if seqs, ok := categorySeqMap[category]; ok {
				filter.CategorySeqs = append(filter.CategorySeqs, seqs...)
			}
		}
	}

	if params.Search != "" {
		filter.Search = sanitizeSearch(params.Search)
	}

	if params.GuSeq != "" {
		if n, err := strconv.Atoi(params.GuSeq); err == nil && n > 0 {
			filter.GuSeq = strconv.Itoa(n)
		}
	}

	if params.Geohashes != "" {
		for _, geohash := range strings.Split(params.Geohashes, ",") {
			geohash = strings.TrimSpace(geohash)
			if geohash != "" {
				filter.Geohashes = append(filter.Geohashes, geohash)
			}
		}
	}

	if strings.EqualFold(params.Weekend, "true") {
		filter.WeekendStart, filter.WeekendEnd = getWeekendRange()
	}

	filter.StartDate = strings.TrimSpace(params.StartDate)
	filter.EndDate = strings.TrimSpace(params.EndDate)

	if params.Page != "" {
		if page, err := strconv.Atoi(params.Page); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if params.Limit != "" {
		if limit, err := strconv.Atoi(params.Limit); err == nil && limit > 0 {
			filter.Limit = limit
			if filter.Limit > 300 {
				filter.Limit = 300
			}
		}
	}

	return filter
}

func (s *EventService) ToSeoulEvent(row model.EventRow) model.SeoulEvent {
	event := model.SeoulEvent{
		ID:       row.EventID,
		Title:    row.EventName,
		Category: mapCategorySeqToCategory(row.CategorySeq),
	}

	if row.StartDate != nil {
		event.StartDate = *row.StartDate
	}
	if row.EndDate != nil {
		event.EndDate = *row.EndDate
	}

	event.Description = sanitizeNull(row.Describe)

	place := sanitizeNull(row.Place)
	orgName := sanitizeNull(row.OrgName)
	switch {
	case place != nil:
		event.LocationName = *place
	case orgName != nil:
		event.LocationName = *orgName
	default:
		event.LocationName = "장소 정보 없음"
	}

	if row.Latitude != nil {
		event.Latitude = *row.Latitude
	}
	if row.Longitude != nil {
		event.Longitude = *row.Longitude
	}

	event.ThumbnailURL = sanitizeNull(row.MainImg)

	homepage := sanitizeNull(row.HomepageLink)
	detailURL := sanitizeNull(row.DetailURL)
	switch {
	case homepage != nil:
		event.ExternalLink = homepage
	case detailURL != nil:
		event.ExternalLink = detailURL
	}

	event.IsFree = row.IsFree
	event.TicketPrice = sanitizeNull(row.TicketPrice)
	event.UseTarget = sanitizeNull(row.UseTarget)
	event.Player = sanitizeNull(row.Player)
	event.OrgName = sanitizeNull(row.OrgName)
	event.Theme = sanitizeNull(row.Theme)
	event.EtcDescription = sanitizeNull(row.EtcDesc)
	event.InquiryNumber = sanitizeNull(row.InquiryNumber)
	event.DisplayTime = sanitizeNull(row.DisplayTime)

	return event
}

func (s *EventService) ToSeoulEvents(rows []model.EventRow) []model.SeoulEvent {
	events := make([]model.SeoulEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, s.ToSeoulEvent(row))
	}

	return events
}

func (s *EventService) GetEvents(params EventQueryParams) (*model.EventListResponse, error) {
	filter := s.BuildFilter(params)
	rows, totalCount, err := s.repo.FindEvents(filter)
	if err != nil {
		return nil, err
	}

	return &model.EventListResponse{
		Events:     s.ToSeoulEvents(rows),
		TotalCount: totalCount,
		Page:       filter.Page,
		Limit:      filter.Limit,
	}, nil
}

func (s *EventService) GetEventByID(id string) (*model.SeoulEvent, error) {
	row, err := s.repo.FindEventByID(id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}

	event := s.ToSeoulEvent(*row)
	return &event, nil
}
