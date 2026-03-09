# seoulful-server-go 구현 플랜

## 개요

Next.js API Routes로 구현된 프론트엔드(`seoulful-client`)의 서버 사이드 로직을 Go 독립 서버로 분리하는 프로젝트.

## 기술 스택

| 항목          | 선택                             | 비고                  |
| ------------- | -------------------------------- | --------------------- |
| 프레임워크    | `gin-gonic/gin`                  | HTTP 라우팅, 미들웨어 |
| DB 클라이언트 | `supabase-community/supabase-go` | PostgREST API 기반    |
| 환경변수      | `joho/godotenv`                  | `.env` 파일 로딩      |

---

## 프로젝트 구조

```
seoulful-server-go/
├── cmd/
│   └── server/
│       └── main.go                 # 엔트리포인트
├── internal/
│   ├── config/
│   │   └── config.go               # 환경변수 로딩 및 설정 구조체
│   ├── handler/
│   │   ├── health.go               # GET /api/health
│   │   └── event.go                # GET /api/events, GET /api/events/:id
│   ├── model/
│   │   ├── db.go                   # DB 테이블 매핑 구조체 (snake_case)
│   │   └── response.go             # API 응답 구조체 (camelCase JSON)
│   ├── repository/
│   │   └── event.go                # Supabase PostgREST 쿼리 로직
│   ├── service/
│   │   └── event.go                # 비즈니스 로직 (매핑, 변환, 필터 빌드)
│   └── middleware/
│       └── cors.go                 # CORS 미들웨어
├── .env.example
├── .gitignore
├── go.mod
└── go.sum
```

---

## 환경변수

```env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your-anon-key
PORT=8080
GIN_MODE=debug
CORS_ORIGIN=http://localhost:3000
```

---

## API 엔드포인트 명세

### 1. `GET /api/health`

**Response 200:**

```json
{ "status": "UP", "timestamp": "2026-03-09T12:00:00Z" }
```

### 2. `GET /api/events`

**Query Parameters:**

| 파라미터    | 타입   | 기본값 | 설명                                              |
| ----------- | ------ | ------ | ------------------------------------------------- |
| `category`  | string | -      | 쉼표 구분 (FESTIVAL,EXHIBITION,PERFORMANCE,OTHER) |
| `search`    | string | -      | 이벤트명/기관명 ILIKE 검색 (최대 100자)           |
| `weekend`   | string | -      | `"true"`이면 금 0시 ~ 일 23:59:59 필터            |
| `startDate` | string | -      | ISO 날짜, `end_date >= startDate`                 |
| `endDate`   | string | -      | ISO 날짜, `start_date <= endDate`                 |
| `guSeq`     | int    | -      | 구 번호 (양수)                                    |
| `geohashes` | string | -      | 쉼표 구분 geohash prefix 리스트                   |
| `page`      | int    | 1      | 페이지 번호                                       |
| `limit`     | int    | 20     | 페이지당 개수 (최대 300)                          |

**Response 200:**

```json
{
  "events": [SeoulEvent, ...],
  "totalCount": 150,
  "page": 1,
  "limit": 20
}
```

### 3. `GET /api/events/:id`

**Response 200:** `SeoulEvent` 객체
**Response 404:** `{ "error": "Event not found" }`
**Response 500:** `{ "error": "Internal server error" }`

---

## 구현 상세

### Phase 1: 프로젝트 초기화

```bash
go mod init seoulful-server-go
go get github.com/gin-gonic/gin
go get github.com/joho/godotenv
go get github.com/supabase-community/supabase-go
```

> **참고**: `supabase-go`는 내부적으로 `postgrest-go`에 의존함. `OrderOpts` 등의 타입을 직접 참조해야 하므로 코드에서 `import postgrest "github.com/supabase-community/postgrest-go"`가 필요할 수 있음.

---

### Phase 2: 설정 및 미들웨어

#### `internal/config/config.go`

```go
package config

import (
    "fmt"
    "os"

    "github.com/joho/godotenv"
)

type Config struct {
    SupabaseURL string
    SupabaseKey string
    Port        string
    GinMode     string
    CORSOrigin  string
}

func Load() (*Config, error) {
    _ = godotenv.Load() // .env 없으면 무시 (프로덕션에서는 OS 환경변수)

    cfg := &Config{
        SupabaseURL: os.Getenv("SUPABASE_URL"),
        SupabaseKey: os.Getenv("SUPABASE_KEY"),
        Port:        os.Getenv("PORT"),
        GinMode:     os.Getenv("GIN_MODE"),
        CORSOrigin:  os.Getenv("CORS_ORIGIN"),
    }

    if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
        return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_KEY are required")
    }
    if cfg.Port == "" {
        cfg.Port = "8080"
    }
    if cfg.GinMode == "" {
        cfg.GinMode = "debug"
    }
    if cfg.CORSOrigin == "" {
        cfg.CORSOrigin = "*"
    }

    return cfg, nil
}
```

#### `internal/middleware/cors.go`

```go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func CORS(allowOrigin string) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", allowOrigin)
        c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
        c.Header("Access-Control-Max-Age", "86400")

        if c.Request.Method == http.MethodOptions {
            c.AbortWithStatus(http.StatusNoContent)
            return
        }
        c.Next()
    }
}
```

---

### Phase 3: 모델 정의

#### `internal/model/db.go`

DB `events` 테이블의 row를 나타내는 구조체. JSON 태그는 DB 컬럼명(snake_case)과 일치.
PostgREST가 `null`을 JSON `null`로 반환하므로 nullable 필드는 포인터 타입 사용.

```go
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
```

#### `internal/model/response.go`

프론트엔드에 반환하는 API 응답 구조체. JSON 태그는 camelCase.

```go
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
```

---

### Phase 4: Repository 계층 (Supabase 쿼리)

#### `internal/repository/event.go`

이 파일이 가장 복잡한 로직. PostgREST 쿼리 빌더로 동적 필터를 구성.

```go
package repository

import (
    "encoding/json"
    "fmt"
    "strings"

    supabase "github.com/supabase-community/supabase-go"
    postgrest "github.com/supabase-community/postgrest-go"
    "seoulful-server-go/internal/model"
)

type EventRepository struct {
    client *supabase.Client
}

func NewEventRepository(client *supabase.Client) *EventRepository {
    return &EventRepository{client: client}
}

// EventFilter는 이벤트 목록 조회 시 사용하는 필터 파라미터
type EventFilter struct {
    CategorySeqs []string  // category_seq IN 필터 (문자열 슬라이스)
    Search       string    // 검색어
    GuSeq        string    // 구 번호 (문자열)
    Geohashes    []string  // geohash prefix 리스트
    WeekendStart string    // 금요일 00:00:00 ISO8601
    WeekendEnd   string    // 일요일 23:59:59 ISO8601
    StartDate    string    // 시작일 필터
    EndDate      string    // 종료일 필터
    Page         int
    Limit        int
}

func (r *EventRepository) FindEvents(filter EventFilter) ([]model.EventRow, int64, error) {
    query := r.client.From("events").Select("*", "exact", false)

    // category_seq IN
    if len(filter.CategorySeqs) > 0 {
        query = query.In("category_seq", filter.CategorySeqs)
    }

    // search (OR: event_name ILIKE OR org_name ILIKE)
    if filter.Search != "" {
        orFilter := fmt.Sprintf(
            "event_name.ilike.%%%s%%,org_name.ilike.%%%s%%",
            filter.Search, filter.Search,
        )
        query = query.Or(orFilter, "")
    }

    // gu_seq EQ
    if filter.GuSeq != "" {
        query = query.Eq("gu_seq", filter.GuSeq)
    }

    // geohash prefix (OR: geohash.like.abc%, geohash.like.def%)
    if len(filter.Geohashes) > 0 {
        var parts []string
        for _, hash := range filter.Geohashes {
            parts = append(parts, fmt.Sprintf("geohash.like.%s%%", hash))
        }
        query = query.Or(strings.Join(parts, ","), "")
    }

    // weekend 필터: start_date <= sundayEnd AND end_date >= fridayStart
    if filter.WeekendStart != "" && filter.WeekendEnd != "" {
        query = query.Lte("start_date", filter.WeekendEnd)
        query = query.Gte("end_date", filter.WeekendStart)
    }

    // startDate: end_date >= startDate
    if filter.StartDate != "" {
        query = query.Gte("end_date", filter.StartDate)
    }

    // endDate: start_date <= endDate
    if filter.EndDate != "" {
        query = query.Lte("start_date", filter.EndDate)
    }

    // 정렬: start_date DESC, event_id DESC
    query = query.Order("start_date", &postgrest.OrderOpts{Ascending: false})
    query = query.Order("event_id", &postgrest.OrderOpts{Ascending: false})

    // 페이지네이션 (0-based, inclusive)
    offset := (filter.Page - 1) * filter.Limit
    query = query.Range(offset, offset+filter.Limit-1, "")

    // 실행
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
    query := r.client.From("events").
        Select("*", "", false).
        Eq("event_id", eventID)

    data, _, err := query.Execute()
    if err != nil {
        return nil, fmt.Errorf("supabase query failed: %w", err)
    }

    var events []model.EventRow
    if err := json.Unmarshal(data, &events); err != nil {
        return nil, fmt.Errorf("failed to unmarshal event: %w", err)
    }

    if len(events) == 0 {
        return nil, nil // 404 처리를 위해 nil 반환
    }

    return &events[0], nil
}
```

**postgrest-go API 참조:**

| 메서드      | 시그니처                              | 비고                                    |
| ----------- | ------------------------------------- | --------------------------------------- |
| `Select`    | `(columns, count string, head bool)`  | count에 `"exact"` 전달하면 총 건수 반환 |
| `Eq`        | `(column, value string)`              | 모든 값이 string 타입                   |
| `In`        | `(column string, values []string)`    | 숫자도 문자열로 변환                    |
| `Or`        | `(filters, foreignTable string)`      | foreignTable은 빈 문자열 `""`           |
| `Lte`/`Gte` | `(column, value string)`              | 비교 연산                               |
| `Order`     | `(column string, opts *OrderOpts)`    | `Ascending: false`가 DESC               |
| `Range`     | `(from, to int, foreignTable string)` | 0-based inclusive                       |
| `Execute`   | `() ([]byte, int64, error)`           | 두 번째 값이 count                      |

**Or 필터 문법:**

- 형식: `"컬럼.연산자.값,컬럼.연산자.값"`
- 예: `"event_name.ilike.%축제%,org_name.ilike.%축제%"`
- 예: `"geohash.like.wydm%,geohash.like.wydn%"`

---

### Phase 5: Service 계층 (비즈니스 로직)

#### `internal/service/event.go`

```go
package service

import (
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
```

**카테고리 매핑 (양방향):**

```go
// 프론트엔드 카테고리 → DB category_seq 목록 (필터 입력용)
var categorySeqMap = map[string][]string{
    "FESTIVAL":    {"9", "10", "11", "12", "13"},
    "EXHIBITION":  {"8"},
    "PERFORMANCE": {"2", "3", "4", "5", "6", "7", "14", "15"},
    "OTHER":       {"1", "16"},
}

// DB category_seq → 프론트엔드 카테고리 (응답 변환용)
var seqToCategoryMap = map[int]model.EventCategory{
    9: model.CategoryFestival, 10: model.CategoryFestival,
    11: model.CategoryFestival, 12: model.CategoryFestival,
    13: model.CategoryFestival,
    8: model.CategoryExhibition,
    2: model.CategoryPerformance, 3: model.CategoryPerformance,
    4: model.CategoryPerformance, 5: model.CategoryPerformance,
    6: model.CategoryPerformance, 7: model.CategoryPerformance,
    14: model.CategoryPerformance, 15: model.CategoryPerformance,
    1: model.CategoryOther, 16: model.CategoryOther,
}

func mapCategorySeqToCategory(seq int) model.EventCategory {
    if cat, ok := seqToCategoryMap[seq]; ok {
        return cat
    }
    return model.CategoryOther
}
```

**sanitizeNull:**

```go
// "NULL", "null", 빈 문자열 → nil 변환
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
```

**검색어 정제:**

```go
var reservedCharsRegex = regexp.MustCompile(`[(),]`)

func sanitizeSearch(search string) string {
    s := strings.TrimSpace(search)
    if utf8.RuneCountInString(s) > 100 {
        runes := []rune(s)
        s = string(runes[:100])
    }
    s = reservedCharsRegex.ReplaceAllString(s, " ")
    return s
}
```

**주말 계산 (Asia/Seoul 기준):**

```go
// 금/토/일이면 현재 주말, 월~목이면 다음 주말
func getWeekendRange() (fridayStart, sundayEnd string) {
    loc, _ := time.LoadLocation("Asia/Seoul")
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

    friday := time.Date(now.Year(), now.Month(), now.Day()+daysUntilFriday,
        0, 0, 0, 0, loc)
    sunday := time.Date(friday.Year(), friday.Month(), friday.Day()+2,
        23, 59, 59, 0, loc)

    return friday.Format(time.RFC3339), sunday.Format(time.RFC3339)
}
```

> **주의**: Docker alpine 이미지에서 `time.LoadLocation` 실패 방지를 위해
> `import _ "time/tzdata"` 추가 필요.

**쿼리 파라미터 → 필터 변환:**

```go
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

func (s *EventService) BuildFilter(params EventQueryParams) repository.EventFilter {
    filter := repository.EventFilter{Page: 1, Limit: 20}

    // category → category_seq 목록
    if params.Category != "" {
        for _, cat := range strings.Split(params.Category, ",") {
            cat = strings.TrimSpace(strings.ToUpper(cat))
            if seqs, ok := categorySeqMap[cat]; ok {
                filter.CategorySeqs = append(filter.CategorySeqs, seqs...)
            }
        }
    }

    // search 정제
    if params.Search != "" {
        filter.Search = sanitizeSearch(params.Search)
    }

    // guSeq (양수 정수만)
    if params.GuSeq != "" {
        if n, err := strconv.Atoi(params.GuSeq); err == nil && n > 0 {
            filter.GuSeq = params.GuSeq
        }
    }

    // geohashes 파싱
    if params.Geohashes != "" {
        for _, h := range strings.Split(params.Geohashes, ",") {
            h = strings.TrimSpace(h)
            if h != "" {
                filter.Geohashes = append(filter.Geohashes, h)
            }
        }
    }

    // weekend 필터
    if strings.ToLower(params.Weekend) == "true" {
        filter.WeekendStart, filter.WeekendEnd = getWeekendRange()
    }

    filter.StartDate = params.StartDate
    filter.EndDate = params.EndDate

    // page
    if params.Page != "" {
        if p, err := strconv.Atoi(params.Page); err == nil && p > 0 {
            filter.Page = p
        }
    }

    // limit (최대 300)
    if params.Limit != "" {
        if l, err := strconv.Atoi(params.Limit); err == nil && l > 0 {
            filter.Limit = l
            if filter.Limit > 300 {
                filter.Limit = 300
            }
        }
    }

    return filter
}
```

**DB Row → API Response 변환:**

```go
func (s *EventService) ToSeoulEvent(row model.EventRow) model.SeoulEvent {
    event := model.SeoulEvent{
        ID:       row.EventID,
        Title:    row.EventName,
        Category: mapCategorySeqToCategory(row.CategorySeq),
    }

    // startDate, endDate
    if row.StartDate != nil { event.StartDate = *row.StartDate }
    if row.EndDate != nil   { event.EndDate = *row.EndDate }

    // description
    event.Description = sanitizeNull(row.Describe)

    // locationName: place → org_name → "장소 정보 없음"
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

    // latitude, longitude
    if row.Latitude != nil  { event.Latitude = *row.Latitude }
    if row.Longitude != nil { event.Longitude = *row.Longitude }

    // thumbnailUrl
    event.ThumbnailURL = sanitizeNull(row.MainImg)

    // externalLink: homepage_link → detail_url
    homepage := sanitizeNull(row.HomepageLink)
    detailURL := sanitizeNull(row.DetailURL)
    switch {
    case homepage != nil:
        event.ExternalLink = homepage
    case detailURL != nil:
        event.ExternalLink = detailURL
    }

    // 나머지 필드
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
```

**공개 메서드:**

```go
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
        return nil, nil // 404
    }
    event := s.ToSeoulEvent(*row)
    return &event, nil
}
```

---

### Phase 6: Handler 계층

#### `internal/handler/health.go`

```go
package handler

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "seoulful-server-go/internal/model"
)

func HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, model.HealthResponse{
        Status:    "UP",
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    })
}
```

#### `internal/handler/event.go`

```go
package handler

import (
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
    "seoulful-server-go/internal/model"
    "seoulful-server-go/internal/service"
)

type EventHandler struct {
    service *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
    return &EventHandler{service: svc}
}

func (h *EventHandler) GetEvents(c *gin.Context) {
    params := service.EventQueryParams{
        Category:  c.Query("category"),
        Search:    c.Query("search"),
        Weekend:   c.Query("weekend"),
        StartDate: c.Query("startDate"),
        EndDate:   c.Query("endDate"),
        GuSeq:     c.Query("guSeq"),
        Geohashes: c.Query("geohashes"),
        Page:      c.Query("page"),
        Limit:     c.Query("limit"),
    }

    result, err := h.service.GetEvents(params)
    if err != nil {
        log.Printf("Error fetching events: %v", err)
        c.JSON(http.StatusInternalServerError, model.ErrorResponse{
            Error: "Internal server error",
        })
        return
    }

    c.JSON(http.StatusOK, result)
}

func (h *EventHandler) GetEventByID(c *gin.Context) {
    id := c.Param("id")

    event, err := h.service.GetEventByID(id)
    if err != nil {
        log.Printf("Error fetching event %s: %v", id, err)
        c.JSON(http.StatusInternalServerError, model.ErrorResponse{
            Error: "Internal server error",
        })
        return
    }

    if event == nil {
        c.JSON(http.StatusNotFound, model.ErrorResponse{
            Error: "Event not found",
        })
        return
    }

    c.JSON(http.StatusOK, event)
}
```

---

### Phase 7: 엔트리포인트

#### `cmd/server/main.go`

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    supabase "github.com/supabase-community/supabase-go"

    "seoulful-server-go/internal/config"
    "seoulful-server-go/internal/handler"
    "seoulful-server-go/internal/middleware"
    "seoulful-server-go/internal/repository"
    "seoulful-server-go/internal/service"
)

func main() {
    // 설정 로드
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Supabase 클라이언트 초기화
    client, err := supabase.NewClient(cfg.SupabaseURL, cfg.SupabaseKey, nil)
    if err != nil {
        log.Fatalf("Failed to initialize Supabase client: %v", err)
    }

    // 의존성 조립: Repository → Service → Handler
    eventRepo := repository.NewEventRepository(client)
    eventSvc := service.NewEventService(eventRepo)
    eventHandler := handler.NewEventHandler(eventSvc)

    // Gin 설정
    gin.SetMode(cfg.GinMode)
    r := gin.Default()
    r.Use(middleware.CORS(cfg.CORSOrigin))

    // 라우트 등록
    api := r.Group("/api")
    {
        api.GET("/health", handler.HealthCheck)
        api.GET("/events", eventHandler.GetEvents)
        api.GET("/events/:id", eventHandler.GetEventByID)
    }

    // 서버 시작
    log.Printf("Server starting on port %s", cfg.Port)
    if err := r.Run(":" + cfg.Port); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
```

---

### Phase 8: 기타 파일

#### `.gitignore`

```
.env
bin/
*.exe
*.test
*.out
vendor/
```

#### `.env.example`

```
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_KEY=your-anon-key
PORT=8080
GIN_MODE=debug
CORS_ORIGIN=http://localhost:3000
```

---

## 구현 순서

| 순서 | 작업                       | 파일                           |
| ---- | -------------------------- | ------------------------------ |
| 1    | go mod init, 의존성 설치   | `go.mod`                       |
| 2    | 설정 로딩                  | `internal/config/config.go`    |
| 3    | CORS 미들웨어              | `internal/middleware/cors.go`  |
| 4    | DB 모델 구조체             | `internal/model/db.go`         |
| 5    | API 응답 구조체            | `internal/model/response.go`   |
| 6    | Repository (Supabase 쿼리) | `internal/repository/event.go` |
| 7    | Service (비즈니스 로직)    | `internal/service/event.go`    |
| 8    | Health 핸들러              | `internal/handler/health.go`   |
| 9    | Event 핸들러               | `internal/handler/event.go`    |
| 10   | 엔트리포인트               | `cmd/server/main.go`           |
| 11   | 기타 파일                  | `.gitignore`, `.env.example`   |

---

## 주의사항

1. **postgrest-go import**: `OrderOpts` 등 사용 시 `import postgrest "github.com/supabase-community/postgrest-go"` 필요
2. **모든 필터 값은 string**: `Eq`, `In`, `Lte`, `Gte` 등의 값 파라미터가 전부 `string`
3. **Range 페이지네이션**: 0-based inclusive, `Range((page-1)*limit, (page-1)*limit+limit-1, "")`
4. **Execute 반환값**: `([]byte, int64, error)` — 두 번째가 count, JSON 직접 unmarshal 필요
5. **타임존**: Docker 환경에서 `import _ "time/tzdata"` 추가 또는 `tzdata` 패키지 설치

---

## 검증 방법

```bash
# 빌드
go build ./cmd/server

# 헬스체크
curl http://localhost:8080/api/health

# 이벤트 목록 (기본)
curl "http://localhost:8080/api/events"

# 이벤트 목록 (필터)
curl "http://localhost:8080/api/events?category=FESTIVAL,EXHIBITION&search=서울&page=1&limit=10"

# 주말 필터
curl "http://localhost:8080/api/events?weekend=true"

# geohash 필터
curl "http://localhost:8080/api/events?geohashes=wydm,wydn"

# 단건 조회
curl http://localhost:8080/api/events/12345

# 404 확인
curl http://localhost:8080/api/events/999999
```

**응답 검증 포인트:**

- `events`, `totalCount`, `page`, `limit` 필드 존재 여부
- JSON 키가 camelCase (`startDate`, `locationName`, `thumbnailUrl`)
- `"NULL"` 문자열이 JSON에서 필드 누락(omitempty)으로 처리
- category가 문자열 enum (`"FESTIVAL"`, `"EXHIBITION"`, `"PERFORMANCE"`, `"OTHER"`)
- 404 응답: `{"error": "Event not found"}`
