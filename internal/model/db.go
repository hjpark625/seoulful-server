package model

type EventRow struct {
	EventID       int      `json:"event_id"`
	CategorySeq   int      `json:"category_seq"`
	GuSeq         int      `json:"gu_seq"`
	EventName     string   `json:"event_name"`
	Period        *string  `json:"period"`
	Place         *string  `json:"place"`
	OrgName       *string  `json:"org_name"`
	UseTarget     *string  `json:"use_target"`
	TicketPrice   *string  `json:"ticket_price"`
	Player        *string  `json:"player"`
	Describe      *string  `json:"describe"`
	EtcDesc       *string  `json:"etc_desc"`
	HomepageLink  *string  `json:"homepage_link"`
	MainImg       *string  `json:"main_img"`
	RegDate       *string  `json:"reg_date"`
	IsPublic      *bool    `json:"is_public"`
	StartDate     *string  `json:"start_date"`
	EndDate       *string  `json:"end_date"`
	Theme         *string  `json:"theme"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	IsFree        *bool    `json:"is_free"`
	DetailURL     *string  `json:"detail_url"`
	Geohash       *string  `json:"geohash"`
	InquiryNumber *string  `json:"inquiry_number"`
	DisplayTime   *string  `json:"display_time"`
}
