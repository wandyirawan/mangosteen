package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"mangosteen/config"
	"mangosteen/internal/admin"
	"mangosteen/internal/auth"
	"mangosteen/internal/crown"
	"mangosteen/internal/db"
	"mangosteen/internal/health"
	"mangosteen/internal/middleware"
	"mangosteen/internal/user"
	"mangosteen/pkg/cache"
	"mangosteen/pkg/crypto"
	"mangosteen/pkg/logger"
	"mangosteen/pkg/queue"
	"mangosteen/pkg/worker"

	"github.com/gofiber/template/html/v2"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	os.MkdirAll(cfg.Logger.BasePath, 0755)
	os.MkdirAll("./data", 0755)

	database := db.MustOpen(cfg.Database.Path)
	defer database.Close()

	// Bootstrap default admin user if configured
	if cfg.Admin.Email != "" && cfg.Admin.Password != "" {
		authRepo := auth.NewRepository(database)
		ctx := context.Background()
		existing, _ := authRepo.FindByEmail(ctx, cfg.Admin.Email)
		if existing == nil {
			now := time.Now().Format(time.RFC3339)
			hash, err := crypto.NewPasswordHasher().Hash(cfg.Admin.Password)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to hash admin password")
			}
			adminUser := &auth.User{
				ID:           uuid.New().String(),
				Email:        cfg.Admin.Email,
				PasswordHash: hash,
				Role:         "admin",
				Active:       1,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := authRepo.Create(ctx, adminUser); err != nil {
				log.Fatal().Err(err).Msg("Failed to create admin user")
			}
			log.Info().Str("email", cfg.Admin.Email).Msg("Created default admin user")
		} else {
			log.Info().Str("email", cfg.Admin.Email).Msg("Admin user already exists")
		}
	}

	// Bootstrap default admin user if configured
	if cfg.Admin.Email != "" && cfg.Admin.Password != "" {
		authRepo := auth.NewRepository(database)
		ctx := context.Background()
		existing, _ := authRepo.FindByEmail(ctx, cfg.Admin.Email)
		if existing == nil {
			now := time.Now().Format(time.RFC3339)
			hash, err := crypto.NewPasswordHasher().Hash(cfg.Admin.Password)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to hash admin password")
			}
			adminUser := &auth.User{
				ID:           uuid.New().String(),
				Email:        cfg.Admin.Email,
				PasswordHash: hash,
				Role:         "admin",
				Active:       1,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := authRepo.Create(ctx, adminUser); err != nil {
				log.Fatal().Err(err).Msg("Failed to create admin user")
			}
			log.Info().Str("email", cfg.Admin.Email).Msg("Created default admin user")
		} else {
			log.Info().Str("email", cfg.Admin.Email).Msg("Admin user already exists")
		}
	}

	var uploadWorker *worker.UploadWorker
	var q *queue.Queue
	if cfg.Worker.Enabled {
		var err error
		q, err = queue.New("./data/queue.db")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize queue")
		}
		defer q.Close()

		workerCfg := worker.Config{
			Endpoint:      cfg.Garage.Endpoint,
			AccessKey:     cfg.Garage.AccessKey,
			SecretKey:     cfg.Garage.SecretKey,
			Bucket:        cfg.Garage.Bucket,
			Region:        cfg.Garage.Region,
			UseSSL:        cfg.Garage.UseSSL,
			CheckInterval: time.Duration(cfg.Worker.CheckInterval) * time.Minute,
			MaxRetries:    cfg.Worker.MaxRetries,
			UploadTimeout: time.Duration(cfg.Worker.UploadTimeout) * time.Second,
			S3Prefix:      cfg.Worker.S3Prefix,
		}

		uploadWorker, err = worker.NewUploadWorker(workerCfg, q)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize upload worker")
		}
	}

	onRotate := func(oldFile string) {
		if !cfg.Worker.Enabled {
			return
		}
		log.Info().Str("file", oldFile).Msg("Log file rotated, adding to upload queue")
		if _, err := q.Enqueue(oldFile); err != nil {
			log.Error().Err(err).Str("file", oldFile).Msg("Failed to enqueue rotated file")
		}
	}

	loggerCfg := logger.Config{
		BasePath: cfg.Logger.BasePath,
		Level:   cfg.Logger.Level,
		Console: cfg.Logger.Console,
		OnRotate: onRotate,
	}

	appLogger, err := logger.New(loggerCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}
	defer appLogger.Close()

	if cfg.Worker.Enabled {
		uploadWorker.Start()
		defer uploadWorker.Stop()
	}

	var valkey *cache.ValkeyClient
	if cfg.Cache.Enabled {
		valkey = cache.NewValkeyClient(cfg.Cache.Addr, cfg.Cache.Password, cfg.Cache.DB)
	}

	jwtManager, err := auth.NewJWTManager(&cfg.JWT)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize JWT manager")
	}

	authRepo := auth.NewRepository(database)
	authService := auth.NewService(authRepo, valkey, jwtManager)
	authHandler := auth.NewHandler(authService, jwtManager)

	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	userRepo := user.NewRepository(database)
	userService := user.NewService(userRepo, valkey)
	userHandler := user.NewHandler(userService)

	healthHandler := health.NewHandler(database, valkey)

	adminHandler := admin.NewHandler(uploadWorker, database)

	crownConsole := crown.New(userService, authService, jwtManager)

	// crownConsole routes moved directly below

	engine := html.New("./templates", ".html")

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Error().Err(err).Msg("HTTP error")
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		},
		Views:       engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	})

	app.Use(recover.New())

	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Str("ip", c.IP()).
			Msg("HTTP request")

		return err
	})

	api := app.Group("/api")

	healthHandler.RegisterRoutes(api)
	authHandler.RegisterRoutes(api)
	adminHandler.RegisterRoutes(api, authMiddleware.RequireAuth(), authMiddleware.RequireAdmin())

	// Self routes (auth required, role user/admin)
	users := api.Group("/users", authMiddleware.RequireAuth())
	users.Get("/me", userHandler.GetMe)
	users.Patch("/me", userHandler.UpdateMe)
	users.Delete("/me", userHandler.DeactivateMe)
	users.Get("/me/attributes", userHandler.GetMyAttributes)
	users.Put("/me/attributes", userHandler.SetMyAttributes)
	users.Delete("/me/attributes/:key", userHandler.DeleteMyAttribute)

	// Admin routes only
	adminUsers := api.Group("/users", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin())
	userHandler.RegisterAdminRoutes(adminUsers)

	// Crown Admin Console
	crown := app.Group("/admin")
	crown.Get("/login", crownConsole.LoginPage)
	crown.Post("/login", crownConsole.Login)
	crown.Get("/logout", crownConsole.Logout)
	crown.Get("/", crownConsole.Dashboard)

	secure := app.Group("/admin", func(c *fiber.Ctx) error {
		return crownConsole.RequireAuth(c)
	})
	secure.Get("/users", crownConsole.UsersList)
	secure.Get("/users/:id", crownConsole.UserDetail)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutting down gracefully...")
		if uploadWorker != nil {
			uploadWorker.Stop()
		}
		appLogger.Close()
		if err := app.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Error during shutdown")
		}
	}()

	log.Info().
		Str("port", cfg.Server.Port).
		Str("environment", cfg.Server.Environment).
		Msg("Starting server")

	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
