package httpapi

import (
	"time"

	"order-playground/apps/gateway-api/internal/httpapi/handlers"
	"order-playground/apps/gateway-api/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(logger gin.HandlerFunc, verifier *middleware.TokenVerifier, handlers *handlers.Handler, ready func() error, timeout time.Duration) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestContext(timeout))
	r.Use(middleware.RequestID())
	r.Use(logger)
	r.Use(middleware.Recovery())
	handlers.Register(r, verifier, ready)
	return r
}
