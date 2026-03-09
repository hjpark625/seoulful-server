package model

type EventCategory string

const (
	CategoryFestival    EventCategory = "FESTIVAL"
	CategoryExhibition  EventCategory = "EXHIBITION"
	CategoryPerformance EventCategory = "PERFORMANCE"
	CategoryOther       EventCategory = "OTHER"
)

type SeoulEvent struct {
	ID             int           `json:"id"`
	Title          string        `json:"title"`
	Description    *string       `json:"description,omitempty"`
	Category       EventCategory `json:"category"`
	StartDate      string        `json:"startDate"`
	EndDate        string        `json:"endDate"`
	LocationName   string        `json:"locationName"`
	Latitude       float64       `json:"latitude"`
	Longitude      float64       `json:"longitude"`
	ThumbnailURL   *string       `json:"thumbnailUrl,omitempty"`
	ExternalLink   *string       `json:"externalLink,omitempty"`
	IsFree         *bool         `json:"isFree,omitempty"`
	TicketPrice    *string       `json:"ticketPrice,omitempty"`
	UseTarget      *string       `json:"useTarget,omitempty"`
	Player         *string       `json:"player,omitempty"`
	OrgName        *string       `json:"orgName,omitempty"`
	Theme          *string       `json:"theme,omitempty"`
	EtcDescription *string       `json:"etcDescription,omitempty"`
	InquiryNumber  *string       `json:"inquiryNumber,omitempty"`
	DisplayTime    *string       `json:"displayTime,omitempty"`
}

type EventListResponse struct {
	Events     []SeoulEvent `json:"events"`
	TotalCount int64        `json:"totalCount"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
