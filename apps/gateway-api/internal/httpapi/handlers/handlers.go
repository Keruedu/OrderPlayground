package handlers

import (
	"net/http"
	"time"

	"order-playground/apps/gateway-api/internal/domain"
	"order-playground/apps/gateway-api/internal/httpapi/middleware"
	"order-playground/apps/gateway-api/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.OrderService
	version string
}

func New(orderService *service.OrderService, version string) *Handler {
	return &Handler{service: orderService, version: version}
}

func (h *Handler) Register(r *gin.Engine, verifier *middleware.TokenVerifier, ready func() error) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := ready(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "gateway-api",
			"version": h.version,
			"time":    time.Now().UTC(),
		})
	})

	api := r.Group("/api")
	api.Use(middleware.Auth(verifier))
	api.Use(middleware.RequireRoles("user", "admin"))
	{
		api.POST("/orders", h.createOrder)
		api.GET("/orders/:id", h.getOrder)
		api.GET("/orders/:id/events", h.getEvents)
		api.POST("/orders/:id/cancel", h.cancelOrder)
	}

	admin := api.Group("/admin")
	admin.Use(middleware.RequireRoles("admin"))
	{
		admin.GET("/orders", h.listOrders)
		admin.GET("/workflows/:workflowId", h.getWorkflow)
	}
}

func (h *Handler) createOrder(c *gin.Context) {
	var req domain.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, _ := middleware.UserFromContext(c.Request.Context())
	resp, err := h.service.CreateOrder(c.Request.Context(), user, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) getOrder(c *gin.Context) {
	order, err := h.service.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *Handler) getEvents(c *gin.Context) {
	events, err := h.service.GetEvents(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *Handler) cancelOrder(c *gin.Context) {
	user, _ := middleware.UserFromContext(c.Request.Context())
	if err := h.service.CancelOrder(c.Request.Context(), c.Param("id"), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "canceled"})
}

func (h *Handler) listOrders(c *gin.Context) {
	orders, err := h.service.ListOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}

func (h *Handler) getWorkflow(c *gin.Context) {
	workflowRun, err := h.service.GetWorkflow(c.Request.Context(), c.Param("workflowId"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, workflowRun)
}
