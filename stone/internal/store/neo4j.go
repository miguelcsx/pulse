package store

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/pulse/stone/internal/config"
)

type GraphStore struct {
	Driver              neo4j.DriverWithContext
	Database            string
	EmbeddingDimensions int
}

func NewNeo4j(cfg *config.Config) (*GraphStore, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4jURI,
		neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	store := &GraphStore{
		Driver:              driver,
		Database:            cfg.Neo4jDatabase,
		EmbeddingDimensions: cfg.EmbeddingDimensions,
	}

	if err := store.EnsureSchema(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("failed to ensure neo4j schema: %w", err)
	}

	slog.Info("connected to Neo4j",
		"database", store.Database,
		"embedding_dimensions", store.EmbeddingDimensions,
	)
	return store, nil
}

func (s *GraphStore) Close(ctx context.Context) error {
	return s.Driver.Close(ctx)
}

func (s *GraphStore) Session(ctx context.Context, mode neo4j.AccessMode) neo4j.SessionWithContext {
	return s.Driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   mode,
		DatabaseName: s.Database,
	})
}

func (s *GraphStore) ExecuteRead(ctx context.Context, fn func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := s.Session(ctx, neo4j.AccessModeRead)
	defer session.Close(ctx)
	return session.ExecuteRead(ctx, fn)
}

func (s *GraphStore) ExecuteWrite(ctx context.Context, fn func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := s.Session(ctx, neo4j.AccessModeWrite)
	defer session.Close(ctx)
	return session.ExecuteWrite(ctx, fn)
}

// RunWrite runs a single write query inside a managed transaction and consumes
// the result so the query is actually executed before the transaction commits.
func (s *GraphStore) RunWrite(ctx context.Context, query string, params map[string]any) error {
	_, err := s.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		_, err = result.Consume(ctx)
		return nil, err
	})
	return err
}

func (s *GraphStore) EnsureSchema(ctx context.Context) error {
	statements := []string{
		"CREATE CONSTRAINT user_id IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE",
		"CREATE CONSTRAINT user_handle IF NOT EXISTS FOR (u:User) REQUIRE u.handle IS UNIQUE",
		"CREATE CONSTRAINT user_email IF NOT EXISTS FOR (u:User) REQUIRE u.email IS UNIQUE",
		"CREATE CONSTRAINT tag_id IF NOT EXISTS FOR (t:Tag) REQUIRE t.id IS UNIQUE",
		"CREATE CONSTRAINT tag_name IF NOT EXISTS FOR (t:Tag) REQUIRE t.name IS UNIQUE",
		"CREATE CONSTRAINT content_id IF NOT EXISTS FOR (c:Content) REQUIRE c.id IS UNIQUE",
		"CREATE CONSTRAINT path_id IF NOT EXISTS FOR (p:Path) REQUIRE p.id IS UNIQUE",
		"CREATE CONSTRAINT room_id IF NOT EXISTS FOR (r:Room) REQUIRE r.id IS UNIQUE",
		"CREATE CONSTRAINT media_asset_id IF NOT EXISTS FOR (m:MediaAsset) REQUIRE m.id IS UNIQUE",
		"CREATE CONSTRAINT refresh_id IF NOT EXISTS FOR (rt:RefreshToken) REQUIRE rt.id IS UNIQUE",
		"CREATE CONSTRAINT event_id IF NOT EXISTS FOR (e:Event) REQUIRE e.id IS UNIQUE",
		"CREATE INDEX room_cluster_key IF NOT EXISTS FOR (r:Room) ON (r.clusterKey)",
		"CREATE INDEX content_created_at IF NOT EXISTS FOR (c:Content) ON (c.createdAt)",
		"CREATE INDEX path_created_at IF NOT EXISTS FOR (p:Path) ON (p.createdAt)",
	}

	vectorIndex := fmt.Sprintf(
		"CREATE VECTOR INDEX content_embedding IF NOT EXISTS FOR (c:Content) ON (c.embedding) OPTIONS {indexConfig: {`vector.dimensions`: %d, `vector.similarity_function`: 'cosine'}}",
		s.EmbeddingDimensions,
	)
	statements = append(statements, vectorIndex)

	_, err := s.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, stmt := range statements {
			result, err := tx.Run(ctx, stmt, nil)
			if err != nil {
				return nil, err
			}
			if _, err = result.Consume(ctx); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}
