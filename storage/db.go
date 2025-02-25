package storage

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client
var collection *mongo.Collection

type Request struct {
	ID          string    `bson:"_id"`
	RequestData string    `bson:"requestData"`
	ResultData  string    `bson:"resultData,omitempty"`
	Status      string    `bson:"status"`
	CreatedAt   time.Time `bson:"createdAt"`
}

func InitDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// Ping
	err = client.Ping(context.Background(), readpref.Primary())
	if err != nil {
		log.Fatal("Ping Failed", err)
	}

	collection = client.Database("websocket-server").Collection("websocket-requests")
	log.Println("Connected to MongoDB")
}

func SaveRequest(id, data string) {
	req := Request{
		ID:          id,
		RequestData: data,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(context.TODO(), req)
	if err != nil {
		log.Println("Error saving request:", err)
	}
}

func SaveResult(id, result string) {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"resultData": result, "status": "done"}}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("Error saving result:", err)
	}
}

func GetResult(id string) (string, bool) {
	var req Request
	err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&req)
	if err != nil {
		return "", false
	}
	return req.ResultData, true
}
