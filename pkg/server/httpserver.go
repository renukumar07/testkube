package server

import (
	"encoding/json"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/problem"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// NewServer returns new HTTP server instance, initializes logger and metrics
func NewServer(config Config) HTTPServer {
	s := HTTPServer{
		Mux:    fiber.New(fiber.Config{DisableStartupMessage: true}),
		Log:    log.DefaultLogger,
		Config: config,
	}

	s.Init()
	return s
}

// HTTPServer represents basic REST HTTP server abstarction
type HTTPServer struct {
	Mux    *fiber.App
	Log    *zap.SugaredLogger
	Routes fiber.Router
	Config Config
}

// Init initializes router and setting up basic routes for health and metrics
func (s *HTTPServer) Init() {

	// global log for requests
	s.Mux.Use(func(c *fiber.Ctx) error {
		s.Log.Debugw("request", "method", string(c.Request().Header.Method()), "path", c.Request().URI().String())
		return c.Next()
	})

	// server generic endpoints
	s.Mux.Get("/health", s.HealthEndpoint())
	s.Mux.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// v1 API
	v1 := s.Mux.Group("/v1")
	v1.Static("/api-docs", "./api/v1")

	s.Routes = v1

}

// Warn writes RFC-7807 json problem to response
func (s *HTTPServer) Warn(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	s.Log.Warnw(err.Error(), "status", status)
	pr := problem.New(status, s.getProblemMessage(err, context))
	return c.JSON(pr)
}

// Error writes RFC-7807 json problem to response
func (s *HTTPServer) Error(c *fiber.Ctx, status int, err error, context ...interface{}) error {
	c.Status(status)
	c.Response().Header.Set("Content-Type", "application/problem+json")
	s.Log.Errorw(err.Error(), "status", status)
	pr := problem.New(status, s.getProblemMessage(err, context))
	return c.JSON(pr)
}

// getProblemMessage creates new JSON based problem message and returns it as string
func (s *HTTPServer) getProblemMessage(err error, context ...interface{}) string {
	message := err.Error()
	if len(context) > 0 {
		b, err := json.Marshal(context[0])
		if err == nil {
			message += ", context: " + string(b)
		}
	}

	return message
}

// Run starts listening for incoming connetions
func (s HTTPServer) Run() error {
	return s.Mux.Listen(s.Config.Addr())
}
