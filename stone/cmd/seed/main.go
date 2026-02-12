package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

const defaultFixturePath = "internal/db/seed/demo/fixture.json"

type seedOptions struct {
	reset       bool
	fixturePath string
}

type demoFixture struct {
	Password  string            `json:"password"`
	Users     []fixtureUser     `json:"users"`
	Tags      []fixtureTag      `json:"tags"`
	Contents  []fixtureContent  `json:"contents"`
	Reactions []fixtureReaction `json:"reactions"`
	Follows   []fixtureFollow   `json:"follows"`
	Rooms     []fixtureRoom     `json:"rooms"`
	Paths     []fixturePath     `json:"paths"`
}

type fixtureUser struct {
	Handle      string `json:"handle"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
}

type fixtureTag struct {
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
}

type fixtureContent struct {
	Key           string   `json:"key"`
	CreatorHandle string   `json:"creator_handle"`
	ContentType   string   `json:"content_type"`
	Body          string   `json:"body"`
	MediaPath     string   `json:"media_path"`
	TagNames      []string `json:"tag_names"`
	HoursAgo      int      `json:"hours_ago"`
}

type fixtureReaction struct {
	UserHandle string `json:"user_handle"`
	ContentKey string `json:"content_key"`
	Kind       string `json:"kind"`
}

type fixtureFollow struct {
	FollowerHandle string `json:"follower_handle"`
	FolloweeHandle string `json:"followee_handle"`
}

type fixtureRoom struct {
	ClusterKey     string   `json:"cluster_key"`
	TagNames       []string `json:"tag_names"`
	MemberHandles  []string `json:"member_handles"`
	ExpiresInHours int      `json:"expires_in_hours"`
}

type fixturePath struct {
	CreatorHandle   string            `json:"creator_handle"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	HoursAgo        int               `json:"hours_ago"`
	Items           []fixturePathItem `json:"items"`
	FollowerHandles []string          `json:"follower_handles"`
}

type fixturePathItem struct {
	ContentKey string `json:"content_key"`
	Note       string `json:"note"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	opts := parseSeedOptions(os.Args[1:])
	fixture, err := loadFixture(opts.fixturePath)
	if err != nil {
		log.Fatalf("failed to load fixture: %v", err)
	}

	db, err := store.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if opts.reset {
		log.Println("Resetting demo data...")
		if err := resetDemoData(db); err != nil {
			log.Fatalf("failed to reset demo data: %v", err)
		}
	}

	if err := seedFromFixture(db, cfg, fixture); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("Seed completed successfully")
}

func parseSeedOptions(args []string) seedOptions {
	opts := seedOptions{
		reset:       false,
		fixturePath: defaultFixturePath,
	}

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch {
		case arg == "--reset":
			opts.reset = true
		case arg == "--fixture" && i+1 < len(args):
			i++
			opts.fixturePath = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--fixture="):
			opts.fixturePath = strings.TrimSpace(strings.TrimPrefix(arg, "--fixture="))
		}
	}

	return opts
}

func loadFixture(path string) (*demoFixture, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return nil, fmt.Errorf("fixture path is required")
	}

	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var fixture demoFixture
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		return nil, err
	}
	if len(fixture.Users) == 0 {
		return nil, fmt.Errorf("fixture must include at least one user")
	}
	if len(fixture.Contents) == 0 {
		return nil, fmt.Errorf("fixture must include at least one content item")
	}
	if strings.TrimSpace(fixture.Password) == "" {
		generated, err := generateRandomPassword(24)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random seed password: %w", err)
		}
		fixture.Password = generated
		log.Printf("No password specified in fixture; generated random seed password: %s", generated)
	}

	return &fixture, nil
}

func resetDemoData(db *gorm.DB) error {
	return db.Exec(`
		TRUNCATE TABLE
			path_follows,
			path_items,
			paths,
			room_members,
			room_tags,
			rooms,
			reactions,
			content_tags,
			contents,
			media_assets,
			events,
			blocks,
			follows,
			user_affinity_edges,
			embeddings,
			refresh_tokens,
			tags,
			users
		RESTART IDENTITY CASCADE
	`).Error
}

func seedFromFixture(db *gorm.DB, cfg *config.Config, fixture *demoFixture) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(fixture.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	now := time.Now().UTC()

	return db.Transaction(func(tx *gorm.DB) error {
		userByHandle := make(map[string]model.User, len(fixture.Users))
		for _, entry := range fixture.Users {
			handle := strings.ToLower(strings.TrimSpace(entry.Handle))
			email := strings.ToLower(strings.TrimSpace(entry.Email))
			if handle == "" || email == "" {
				return fmt.Errorf("user handle and email are required")
			}

			user := model.User{
				Handle:      handle,
				Email:       email,
				Password:    string(passwordHash),
				DisplayName: strings.TrimSpace(entry.DisplayName),
				Bio:         strings.TrimSpace(entry.Bio),
				Location:    strings.TrimSpace(entry.Location),
			}
			if err := tx.
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "handle"}},
					DoUpdates: clause.AssignmentColumns([]string{"email", "password", "display_name", "bio", "location", "updated_at"}),
				}).
				Create(&user).Error; err != nil {
				return fmt.Errorf("failed to upsert user %q: %w", handle, err)
			}
			if err := tx.Where("handle = ?", handle).First(&user).Error; err != nil {
				return fmt.Errorf("failed to load user %q: %w", handle, err)
			}
			userByHandle[handle] = user
		}

		tagByName := make(map[string]model.Tag)
		ensureTag := func(rawName string, usageCount int) (model.Tag, error) {
			name := normalizeTagName(rawName)
			if name == "" {
				return model.Tag{}, fmt.Errorf("tag name cannot be empty")
			}
			if tag, ok := tagByName[name]; ok {
				return tag, nil
			}

			tag := model.Tag{
				Name:       name,
				UsageCount: maxInt(usageCount, 0),
			}
			if err := tx.
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "name"}},
					DoUpdates: clause.Assignments(map[string]any{"usage_count": maxInt(usageCount, 0)}),
				}).
				Create(&tag).Error; err != nil {
				return model.Tag{}, fmt.Errorf("failed to upsert tag %q: %w", name, err)
			}
			if err := tx.Where("name = ?", name).First(&tag).Error; err != nil {
				return model.Tag{}, fmt.Errorf("failed to load tag %q: %w", name, err)
			}

			tagByName[name] = tag
			return tag, nil
		}

		for _, entry := range fixture.Tags {
			if _, err := ensureTag(entry.Name, entry.UsageCount); err != nil {
				return err
			}
		}

		contentByKey := make(map[string]model.Content, len(fixture.Contents))
		for _, entry := range fixture.Contents {
			key := strings.TrimSpace(entry.Key)
			if key == "" {
				return fmt.Errorf("content key is required")
			}
			if _, exists := contentByKey[key]; exists {
				return fmt.Errorf("duplicate content key %q", key)
			}

			creator, ok := userByHandle[strings.ToLower(strings.TrimSpace(entry.CreatorHandle))]
			if !ok {
				return fmt.Errorf("unknown creator_handle %q", entry.CreatorHandle)
			}

			contentType := strings.ToLower(strings.TrimSpace(entry.ContentType))
			if !model.IsValidContentType(contentType) {
				return fmt.Errorf("unsupported content_type %q", entry.ContentType)
			}

			createdAt := now.Add(-time.Duration(maxInt(entry.HoursAgo, 0)) * time.Hour)
			content := model.Content{
				CreatorID:   creator.ID,
				ContentType: contentType,
				Body:        strings.TrimSpace(entry.Body),
				CreatedAt:   createdAt,
				UpdatedAt:   createdAt,
			}

			if contentType != model.ContentTypeText {
				mediaPath := strings.TrimLeft(strings.TrimSpace(entry.MediaPath), "/")
				if mediaPath == "" {
					mediaPath = fmt.Sprintf("demo/%s.jpg", key)
				}
				content.MediaURL = joinURL(cfg.StorageBaseURL, mediaPath)
			}

			if err := tx.Create(&content).Error; err != nil {
				return fmt.Errorf("failed to create content %q: %w", key, err)
			}

			for _, rawTagName := range entry.TagNames {
				tag, err := ensureTag(rawTagName, 0)
				if err != nil {
					return err
				}
				contentTag := model.ContentTag{
					ContentID: content.ID,
					TagID:     tag.ID,
				}
				if err := tx.
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(&contentTag).Error; err != nil {
					return fmt.Errorf("failed to attach tag %q to content %q: %w", rawTagName, key, err)
				}
			}

			contentByKey[key] = content
		}

		for _, entry := range fixture.Reactions {
			user, ok := userByHandle[strings.ToLower(strings.TrimSpace(entry.UserHandle))]
			if !ok {
				return fmt.Errorf("unknown reaction user_handle %q", entry.UserHandle)
			}
			content, ok := contentByKey[strings.TrimSpace(entry.ContentKey)]
			if !ok {
				return fmt.Errorf("unknown reaction content_key %q", entry.ContentKey)
			}
			kind := strings.ToLower(strings.TrimSpace(entry.Kind))
			if !model.IsValidReactionKind(kind) {
				return fmt.Errorf("unsupported reaction kind %q", entry.Kind)
			}

			reaction := model.Reaction{
				UserID:    user.ID,
				ContentID: content.ID,
				Kind:      kind,
				CreatedAt: now,
			}
			if err := tx.
				Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "user_id"}, {Name: "content_id"}, {Name: "kind"}},
					DoNothing: true,
				}).
				Create(&reaction).Error; err != nil {
				return fmt.Errorf("failed to create reaction %q: %w", kind, err)
			}
		}

		for _, entry := range fixture.Follows {
			follower, ok := userByHandle[strings.ToLower(strings.TrimSpace(entry.FollowerHandle))]
			if !ok {
				return fmt.Errorf("unknown follower_handle %q", entry.FollowerHandle)
			}
			followee, ok := userByHandle[strings.ToLower(strings.TrimSpace(entry.FolloweeHandle))]
			if !ok {
				return fmt.Errorf("unknown followee_handle %q", entry.FolloweeHandle)
			}
			if follower.ID == followee.ID {
				continue
			}

			follow := model.Follow{
				FollowerID: follower.ID,
				FolloweeID: followee.ID,
				CreatedAt:  now,
			}
			if err := tx.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&follow).Error; err != nil {
				return fmt.Errorf("failed to create follow %q -> %q: %w", follower.Handle, followee.Handle, err)
			}
		}

		for i, entry := range fixture.Rooms {
			clusterKey := strings.TrimSpace(entry.ClusterKey)
			if clusterKey == "" {
				clusterKey = fmt.Sprintf("demo-room-%d", i+1)
			}

			expiresInHours := maxInt(entry.ExpiresInHours, 24)
			room := model.Room{
				ClusterKey: clusterKey,
				ExpiresAt:  now.Add(time.Duration(expiresInHours) * time.Hour),
				CreatedAt:  now,
			}
			if err := tx.Create(&room).Error; err != nil {
				return fmt.Errorf("failed to create room %q: %w", clusterKey, err)
			}

			for _, rawTag := range entry.TagNames {
				tag, err := ensureTag(rawTag, 0)
				if err != nil {
					return err
				}
				roomTag := model.RoomTag{
					RoomID: room.ID,
					TagID:  tag.ID,
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roomTag).Error; err != nil {
					return fmt.Errorf("failed to add room tag %q: %w", rawTag, err)
				}
			}

			for _, rawHandle := range entry.MemberHandles {
				member, ok := userByHandle[strings.ToLower(strings.TrimSpace(rawHandle))]
				if !ok {
					return fmt.Errorf("unknown room member handle %q", rawHandle)
				}
				roomMember := model.RoomMember{
					RoomID:   room.ID,
					UserID:   member.ID,
					JoinedAt: now,
				}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roomMember).Error; err != nil {
					return fmt.Errorf("failed to add room member %q: %w", rawHandle, err)
				}
			}
		}

		for _, entry := range fixture.Paths {
			creator, ok := userByHandle[strings.ToLower(strings.TrimSpace(entry.CreatorHandle))]
			if !ok {
				return fmt.Errorf("unknown path creator_handle %q", entry.CreatorHandle)
			}
			createdAt := now.Add(-time.Duration(maxInt(entry.HoursAgo, 0)) * time.Hour)
			path := model.Path{
				CreatorID:   creator.ID,
				Title:       strings.TrimSpace(entry.Title),
				Description: strings.TrimSpace(entry.Description),
				CreatedAt:   createdAt,
				UpdatedAt:   createdAt,
			}
			if err := tx.Create(&path).Error; err != nil {
				return fmt.Errorf("failed to create path %q: %w", path.Title, err)
			}

			for idx, item := range entry.Items {
				content, ok := contentByKey[strings.TrimSpace(item.ContentKey)]
				if !ok {
					return fmt.Errorf("unknown path item content_key %q", item.ContentKey)
				}
				pathItem := model.PathItem{
					PathID:    path.ID,
					ContentID: content.ID,
					Position:  idx + 1,
					Note:      strings.TrimSpace(item.Note),
				}
				if err := tx.Create(&pathItem).Error; err != nil {
					return fmt.Errorf("failed to create path item %q: %w", item.ContentKey, err)
				}
			}

			for _, rawHandle := range entry.FollowerHandles {
				follower, ok := userByHandle[strings.ToLower(strings.TrimSpace(rawHandle))]
				if !ok {
					return fmt.Errorf("unknown path follower handle %q", rawHandle)
				}
				pathFollow := model.PathFollow{
					PathID:    path.ID,
					UserID:    follower.ID,
					CreatedAt: now,
				}
				if err := tx.
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(&pathFollow).Error; err != nil {
					return fmt.Errorf("failed to follow path %q: %w", path.Title, err)
				}
			}

			var followerCount int64
			if err := tx.Model(&model.PathFollow{}).Where("path_id = ?", path.ID).Count(&followerCount).Error; err != nil {
				return fmt.Errorf("failed to count followers for path %q: %w", path.Title, err)
			}
			if err := tx.Model(&model.Path{}).Where("id = ?", path.ID).Update("follower_count", int(followerCount)).Error; err != nil {
				return fmt.Errorf("failed to update follower_count for path %q: %w", path.Title, err)
			}
		}

		return nil
	})
}

func normalizeTagName(raw string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "#")))
}

func joinURL(base, rel string) string {
	trimmedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	trimmedRel := strings.TrimLeft(strings.TrimSpace(rel), "/")
	if trimmedBase == "" {
		return "/" + trimmedRel
	}
	return trimmedBase + "/" + trimmedRel
}

func maxInt(v, minValue int) int {
	if v < minValue {
		return minValue
	}
	return v
}

// generateRandomPassword creates a cryptographically random hex-encoded password
// of the given byte length (the resulting string will be 2x that length).
func generateRandomPassword(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
