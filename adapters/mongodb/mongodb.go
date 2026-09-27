package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrNotFound = errors.New("mongodb: document not found")
	ErrInvalid  = errors.New("mongodb: invalid config")
)

type Config struct {
	URI      string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	AuthDB   string
	Params   map[string]string
	Timeout  time.Duration
}

type DB struct {
	client *mongo.Client
	db     *mongo.Database
	name   string
}

func Open(cfg Config) (*DB, error) {
	uri := cfg.URI
	if uri == "" {
		uri = buildURI(cfg)
	}
	if uri == "" || cfg.Database == "" {
		return nil, ErrInvalid
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return &DB{
		client: client,
		db:     client.Database(cfg.Database),
		name:   cfg.Database,
	}, nil
}

func MustOpen(cfg Config) *DB {
	d, err := Open(cfg)
	if err != nil {
		panic(fmt.Errorf("mongodb: %w", err))
	}
	return d
}

func (d *DB) Client() *mongo.Client {
	return d.client
}

func (d *DB) Database() *mongo.Database {
	return d.db
}

func (d *DB) Name() string {
	return d.name
}

func (d *DB) Collection(name string) *Collection {
	return &Collection{col: d.db.Collection(name)}
}

func (d *DB) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

func (d *DB) Close(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}

func (d *DB) ListCollectionNames(ctx context.Context) ([]string, error) {
	return d.db.ListCollectionNames(ctx, bson.D{})
}

func (d *DB) Drop(ctx context.Context) error {
	return d.db.Drop(ctx)
}

func buildURI(cfg Config) string {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	port := cfg.Port
	if port == 0 {
		port = 27017
	}
	auth := ""
	if cfg.User != "" {
		auth = cfg.User
		if cfg.Password != "" {
			auth += ":" + cfg.Password
		}
		auth += "@"
	}
	uri := fmt.Sprintf("mongodb://%s%s:%d", auth, host, port)
	q := ""
	if cfg.AuthDB != "" {
		q = "authSource=" + cfg.AuthDB
	}
	for k, v := range cfg.Params {
		if q != "" {
			q += "&"
		}
		q += k + "=" + v
	}
	if q != "" {
		uri += "/?" + q
	}
	return uri
}
