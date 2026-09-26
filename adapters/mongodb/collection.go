// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collection struct {
	col *mongo.Collection
}

func (c *Collection) Raw() *mongo.Collection {
	return c.col
}

func (c *Collection) Name() string {
	return c.col.Name()
}

func (c *Collection) InsertOne(ctx context.Context, document any) (any, error) {
	res, err := c.col.InsertOne(ctx, document)
	if err != nil {
		return nil, err
	}
	return res.InsertedID, nil
}

func (c *Collection) InsertMany(ctx context.Context, documents []any) ([]any, error) {
	res, err := c.col.InsertMany(ctx, documents)
	if err != nil {
		return nil, err
	}
	return res.InsertedIDs, nil
}

func (c *Collection) FindOne(ctx context.Context, filter any, dest any) error {
	err := c.col.FindOne(ctx, filter).Decode(dest)
	if err == mongo.ErrNoDocuments {
		return ErrNotFound
	}
	return err
}

func (c *Collection) Find(ctx context.Context, filter any, dest any, opts ...FindOption) error {
	findOpts := options.Find()
	cfg := &findConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.limit > 0 {
		findOpts.SetLimit(cfg.limit)
	}
	if cfg.skip > 0 {
		findOpts.SetSkip(cfg.skip)
	}
	if cfg.sort != nil {
		findOpts.SetSort(cfg.sort)
	}
	if cfg.projection != nil {
		findOpts.SetProjection(cfg.projection)
	}
	cursor, err := c.col.Find(ctx, filter, findOpts)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, dest)
}

func (c *Collection) UpdateOne(ctx context.Context, filter any, update any) (int64, error) {
	res, err := c.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

func (c *Collection) UpdateMany(ctx context.Context, filter any, update any) (int64, error) {
	res, err := c.col.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

func (c *Collection) ReplaceOne(ctx context.Context, filter any, replacement any) (int64, error) {
	res, err := c.col.ReplaceOne(ctx, filter, replacement)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

func (c *Collection) DeleteOne(ctx context.Context, filter any) (int64, error) {
	res, err := c.col.DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (c *Collection) DeleteMany(ctx context.Context, filter any) (int64, error) {
	res, err := c.col.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (c *Collection) Count(ctx context.Context, filter any) (int64, error) {
	if filter == nil {
		filter = bson.D{}
	}
	return c.col.CountDocuments(ctx, filter)
}

func (c *Collection) Aggregate(ctx context.Context, pipeline any, dest any) error {
	cursor, err := c.col.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	return cursor.All(ctx, dest)
}

func (c *Collection) EnsureIndex(ctx context.Context, keys any, unique bool) (string, error) {
	model := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	return c.col.Indexes().CreateOne(ctx, model)
}

func (c *Collection) Drop(ctx context.Context) error {
	return c.col.Drop(ctx)
}

type findConfig struct {
	limit      int64
	skip       int64
	sort       any
	projection any
}

type FindOption func(*findConfig)

func Limit(n int64) FindOption {
	return func(c *findConfig) { c.limit = n }
}

func Skip(n int64) FindOption {
	return func(c *findConfig) { c.skip = n }
}

func Sort(sort any) FindOption {
	return func(c *findConfig) { c.sort = sort }
}

func Projection(projection any) FindOption {
	return func(c *findConfig) { c.projection = projection }
}

func Set(fields bson.M) bson.M {
	return bson.M{"$set": fields}
}

func Inc(fields bson.M) bson.M {
	return bson.M{"$inc": fields}
}
