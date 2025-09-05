package http

import (
	"github.com/gin-gonic/gin"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/rs/zerolog"
	"net/http"
)

// Server is a wrapper for the HTTP server.
type Server struct {
	*http.Server
	logger zerolog.Logger
}

// NewServer creates and configures a new Gin server.
func NewServer(cfg *config.Config, handlers *Handlers, logger *zerolog.Logger) *Server {
	log := logger.With().Str("layer", "http_server").Logger()
	log.Info().Msg("initializing http server")

	log.Info().Str("mode", cfg.HTTP.GinMode).Msg("setting gin mode")
	gin.SetMode(cfg.HTTP.GinMode)

	router := gin.New()

	log.Info().Msg("initializing middleware: recovery")
	router.Use(gin.Recovery())

	log.Info().Msg("registering api routes")
	handlers.RegisterRoutes(router)

	log.Info().Msg("registering health check endpoint")
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	server := &http.Server{
		Addr:    cfg.HTTP.Port,
		Handler: router,
	}

	return &Server{server, log}
}
