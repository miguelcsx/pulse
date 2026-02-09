package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
	"github.com/pulse/stone/internal/store"
	"github.com/pulse/stone/internal/ws"
)

type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	redis   *redis.Client
	storage store.Storage
	router  *gin.Engine
	hub     *ws.Hub

	authService     *service.AuthService
	userService     *service.UserService
	contentService  *service.ContentService
	tagService      *service.TagService
	feedService     *service.FeedService
	affinityService *service.AffinityService
	followService   *service.FollowService
	roomService     *service.RoomService
	pathService     *service.PathService
	eventService    *service.EventService
	mediaService    *service.MediaService
}

func NewServer(cfg *config.Config, db *gorm.DB, rdb *redis.Client, storage store.Storage) *Server {
	hub := ws.NewHub()
	go hub.Run()

	tagSvc := service.NewTagService(db)
	mediaSvc := service.NewMediaService(db, storage)

	s := &Server{
		cfg:     cfg,
		db:      db,
		redis:   rdb,
		storage: storage,
		hub:     hub,

		authService:     service.NewAuthService(db, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL),
		userService:     service.NewUserService(db),
		tagService:      tagSvc,
		contentService:  service.NewContentService(db, storage, tagSvc, mediaSvc),
		feedService:     service.NewFeedService(db),
		affinityService: service.NewAffinityService(db),
		followService:   service.NewFollowService(db),
		roomService:     service.NewRoomService(db),
		pathService:     service.NewPathService(db),
		eventService:    service.NewEventService(db),
		mediaService:    mediaSvc,
	}

	if cfg.Env == "production" {
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
	r.Use(middleware.SecureHeaders())
	r.Use(middleware.CORS(s.cfg.CORSOrigins))
	r.Use(middleware.RateLimit(s.redis, s.cfg.RateLimitRPS, s.cfg.RateLimitBurst))

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

			// Social
			protected.GET("/suggestions", s.GetSuggestions)
			protected.POST("/follow/:id", s.FollowUser)
			protected.DELETE("/follow/:id", s.UnfollowUser)
			protected.POST("/block/:id", s.BlockUser)
			protected.DELETE("/block/:id", s.UnblockUser)

			// Rooms
			protected.GET("/rooms", s.ListRooms)
			protected.POST("/rooms/:id/enter", s.EnterRoom)
			protected.POST("/rooms/:id/leave", s.LeaveRoom)

			// Paths
			protected.POST("/paths", s.CreatePath)
			protected.GET("/paths", s.ListPaths)
			protected.GET("/paths/:id", s.GetPath)
			protected.POST("/paths/:id/follow", s.FollowPath)

			// Events
			protected.POST("/events", s.RecordEvents)
		}
	}
}
