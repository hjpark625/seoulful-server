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
		log.Printf("error fetching events: %v", err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *EventHandler) GetEventByID(c *gin.Context) {
	id := c.Param("id")

	event, err := h.service.GetEventByID(id)
	if err != nil {
		log.Printf("error fetching event %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Internal server error"})
		return
	}

	if event == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "Event not found"})
		return
	}

	c.JSON(http.StatusOK, event)
}
