package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"mekari-esign/internal/config"
	"mekari-esign/internal/delivery/http/handler"
)

type Router struct {
	app            *fiber.App
	config         *config.Config
	esignHandler   *handler.EsignHandler
	healthHandler  *handler.HealthHandler
	oauthHandler   *handler.OAuthHandler
	webhookHandler *handler.WebhookHandler
	logHandler     *handler.LogHandler
}

func NewRouter(
	cfg *config.Config,
	esignHandler *handler.EsignHandler,
	healthHandler *handler.HealthHandler,
	oauthHandler *handler.OAuthHandler,
	webhookHandler *handler.WebhookHandler,
	logHandler *handler.LogHandler,
) *Router {
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	return &Router{
		app:            app,
		config:         cfg,
		esignHandler:   esignHandler,
		healthHandler:  healthHandler,
		oauthHandler:   oauthHandler,
		webhookHandler: webhookHandler,
		logHandler:     logHandler,
	}
}

func (r *Router) Setup() *fiber.App {
	// Middleware
	r.app.Use(recover.New())
	r.app.Use(requestid.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	if r.config.IsDevelopment() {
		r.app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
		}))
	}

	// Health check route
	r.app.Get("/health", r.healthHandler.Health)

	// Log viewer route (HTML page)
	r.app.Get("/logs", r.logHandler.LogViewer)

	// OAuth callback route (must be at root level for redirect)
	r.app.Get("/redirect/oauth", r.oauthHandler.OAuthCallback)

	// Webhook routes (at root level for external callbacks)
	r.app.Post("/webhook/mekari", r.webhookHandler.MekariCallback)

	// API v1 routes
	api := r.app.Group("/api/v1")
	{
		// OAuth routes
		oauth := api.Group("/oauth")
		{
			oauth.Get("/check", r.oauthHandler.CheckCode)
			oauth.Get("/authorize", r.oauthHandler.CheckCodeAndRedirect)
			oauth.Post("/save-code", r.oauthHandler.SaveCode)
			oauth.Post("/exchange", r.oauthHandler.ExchangeCode)
			oauth.Post("/refresh", r.oauthHandler.RefreshAccessToken)
			oauth.Get("/token", r.oauthHandler.GetToken)
		}

		// eSign routes
		esign := api.Group("/esign")
		{
			esign.Get("/profile", r.esignHandler.GetProfile)
			esign.Get("/documents", r.esignHandler.GetDocuments)
			esign.Post("/documents/request-sign", r.esignHandler.GlobalRequestSign)
		}

		// Log routes
		logs := api.Group("/logs")
		{
			logs.Get("", r.logHandler.GetLogs)
			logs.Get("/search", r.logHandler.SearchLogs)
		}
	}

	return r.app
}

func (r *Router) GetApp() *fiber.App {
	return r.app
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"message": err.Error(),
		"error": fiber.Map{
			"code":    code,
			"message": err.Error(),
		},
	})
}
