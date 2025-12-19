package repository

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotEnoughStock = errors.New("No enough stock")

type MongoInventoryRepository struct {
	col *mongo.Collection
}

type inventoryDoc struct {
	ProductID string `bson:"product_id"`
	Qty       int32  `bson:"qty"`
}

func NewMongoInventoryRepository(client *mongo.Client, dbName string) *MongoInventoryRepository {
	col := client.Database(dbName).Collection("inventory")
	return &MongoInventoryRepository{col: col}
}

func (r *MongoInventoryRepository) Get(ctx context.Context, productID string) (int32, error) {
	var doc inventoryDoc
	err := r.col.FindOne(ctx, bson.M{"product_id": productID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	return doc.Qty, nil
}

func (r *MongoInventoryRepository) Reserve(ctx context.Context, productID string, qty int32) error {
	filter := bson.M{
		"product_id": productID,
		"qty":        bson.M{"$gte": qty},
	}
	update := bson.M{
		"$inc": bson.M{"qty": -qty},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotEnoughStock
	}
	return nil
}

func (r *MongoInventoryRepository) SetStock(ctx context.Context, productID string, qty int32) error {
	filter := bson.M{"product_id": productID}
	update := bson.M{"$set": bson.M{"qty": qty}}
	_, err := r.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	return nil
}

func ConnectMongo(ctx context.Context, uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}
	return client, nil
}
