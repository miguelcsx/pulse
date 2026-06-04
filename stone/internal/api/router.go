package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
	"github.com/pulse/stone/internal/store"
	"github.com/pulse/stone/internal/ws"
)

type Server struct {
	cfg      *config.Config
	db       *gorm.DB
	redis    *redis.Client
	storage  store.Storage
	router   *gin.Engine
	hub      *ws.Hub
	embedder service.Embedder

	authService    *service.AuthService
	userService    *service.UserService
	contentService *service.ContentService
	tagService     *service.TagService
	feedService    *service.FeedService
	followService  *service.FollowService
	roomService    *service.RoomService
	pathService    *service.PathService
	eventService   *service.EventService
	mediaService   *service.MediaService
	adviceService  *service.AdviceService
}

func NewServer(cfg *config.Config, db *gorm.DB, rdb *redis.Client, storage store.Storage) *Server {
	hub := ws.NewHub()
	go hub.Run()

	tagSvc := service.NewTagService(db, rdb, cfg.TagCacheTTL)
	mediaSvc := service.NewMediaService(db, storage)
	roomSvc := service.NewRoomService(db, cfg)

	var embedder service.Embedder
	if cfg.OllamaBaseURL != "" && cfg.OllamaModel != "" {
		embedder = service.NewOllamaEmbedder(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.EmbeddingDimensions, cfg.OllamaTimeout)
	} else {
		embedder = service.NewHashEmbedder(cfg.EmbeddingDimensions)
	}

	s := &Server{
		cfg:      cfg,
		db:       db,
		redis:    rdb,
		storage:  storage,
		hub:      hub,
		embedder: embedder,

		authService:    service.NewAuthService(db, rdb, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL, cfg.LoginMaxAttempts, cfg.LoginLockoutDuration),
		userService:    service.NewUserService(db),
		tagService:     tagSvc,
		contentService: service.NewContentService(db, storage, tagSvc, mediaSvc, roomSvc),
		feedService:    service.NewFeedService(db, roomSvc, cfg),
		followService:  service.NewFollowService(db),
		roomService:    roomSvc,
		pathService:    service.NewPathService(db),
		eventService:   service.NewEventService(db, cfg),
		mediaService:   mediaSvc,
		adviceService:  service.NewAdviceService(db, embedder),
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(splitCSV(cfg.TrustedProxies)); err != nil {
		// Fallback to local proxies if env config is malformed.
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	}
	s.router = r
	s.setupRoutes()
	return s
}

// MediaService returns the media service for use by the caller (e.g. for
// graceful shutdown and scheduler wiring).
func (s *Server) MediaService() *service.MediaService {
	return s.mediaService
}

// PathService returns the path service for scheduler wiring.
func (s *Server) PathService() *service.PathService {
	return s.pathService
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (s *Server) setupRoutes() {
	r := s.router

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.SecureHeadersWithEnv(s.cfg.Env))
	r.Use(middleware.CORS(s.cfg.CORSOrigins))
	r.Use(middleware.RateLimit(s.redis, s.cfg.RateLimitRPS, s.cfg.RateLimitBurst, s.cfg.RateLimitFailOpen))

	// Prometheus metrics middleware and endpoint
	if s.cfg.MetricsEnabled {
		r.Use(middleware.Metrics())
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// Serve uploaded files
	r.Static(s.cfg.UploadPublicPath, s.cfg.StoragePath)

	// WebSocket endpoint
	r.GET("/ws", s.HandleWebSocket)

	api := r.Group(s.cfg.APIBasePath)
	{
		api.GET("/health", s.Health)

		// Auth (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", s.Register)
			auth.POST("/login", s.Login)
			auth.POST("/demo", s.DemoLogin)
			auth.POST("/refresh", s.Refresh)
			auth.POST("/logout", s.Logout)
		}

		// Tags (public read)
		api.GET("/tags", s.ListTags)
		api.GET("/tags/search", s.SearchTags)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.Auth(s.cfg.JWTSecret))
		{
			// User profile
			protected.GET("/me", s.GetMe)
			protected.PUT("/me", s.UpdateMe)
			protected.GET("/users/search", s.SearchUsers)
			protected.GET("/users/:id", s.GetUser)
			protected.GET("/users/:id/content", s.ListUserContent)

			// Content
			protected.POST("/content", s.CreateContent)
			protected.GET("/content/:id", s.GetContent)
			protected.DELETE("/content/:id", s.DeleteContent)
			protected.POST("/media/uploads/init", s.InitMediaUpload)
			protected.PUT("/media/uploads/:id/file", s.UploadMediaBinary)
			protected.GET("/media/assets/:id", s.GetMediaAsset)

			// Semantic reactions (replaces likes)
			protected.POST("/content/:id/react", s.ReactToContent)
			protected.DELETE("/content/:id/react", s.RemoveReaction)

			// Feed
			protected.GET("/feed", s.GetFeed)
			protected.GET("/today", s.GetToday)

			// Human advice network
			protected.POST("/asks", s.CreateAsk)
			protected.GET("/asks/:id/bridges", s.GetAskBridges)
			protected.PUT("/asks/:id/visibility", s.UpdateAskVisibility)
			protected.POST("/asks/:id/perspective", s.AddPerspective)
			protected.POST("/bridges/:id/ask", s.AskBridge)
			protected.POST("/bridges/:id/respond", s.RespondBridge)
			protected.POST("/bridges/:id/signal", s.SignalBridge)
			protected.GET("/commons", s.ListCommons)
			protected.GET("/network", s.GetNetwork)
			protected.GET("/help-sessions", s.ListHelpSessions)
			protected.POST("/help-sessions/:id/join", s.JoinHelpSession)
			protected.PUT("/me/trust-profile", s.UpdateTrustProfile)

			// Social
			protected.POST("/follow/:id", s.FollowUser)
			protected.DELETE("/follow/:id", s.UnfollowUser)
			protected.POST("/block/:id", s.BlockUser)
			protected.DELETE("/block/:id", s.UnblockUser)

			// Rooms
			protected.GET("/rooms", s.ListRooms)
			protected.GET("/rooms/:id/content", s.GetRoomContent)
			protected.POST("/rooms/:id/enter", s.EnterRoom)
			protected.POST("/rooms/:id/leave", s.LeaveRoom)

			// Paths (auto-generated by scheduler)
			protected.GET("/paths", s.ListPaths)
			protected.GET("/paths/:id", s.GetPath)
			protected.POST("/paths/:id/follow", s.FollowPath)
			protected.DELETE("/paths/:id/follow", s.UnfollowPath)

			// Events
			protected.POST("/events", s.RecordEvents)
		}
	}
}
