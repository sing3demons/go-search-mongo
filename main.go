package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Product struct {
	Id          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Title       string             `json:"title,omitempty" bson:"title,omitempty"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Image       string             `json:"image,omitempty" bson:"image,omitempty"`
	Price       int                `json:"price,omitempty" bson:"price,omitempty"`
}

func InitMongoDB() *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	database := client.Database("go_search")
	return database
}

func main() {

	app := fiber.New()

	app.Use(cors.New())

	app.Post("/api/products/populate", Insert)

	app.Get("/", FindAll)

	app.Get("/api/products/backend", Find)

	app.Listen(":8080")
}

func Find(c *fiber.Ctx) error {
	db := InitMongoDB()
	collection := db.Collection("products")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	var products []Product

	filter := bson.M{}
	findOptions := options.Find()

	if s := c.Query("s"); s != "" {
		filter = bson.M{
			"$or": []bson.M{
				{
					"title": bson.M{
						"$regex": primitive.Regex{
							Pattern: s,
							Options: "i",
						},
					},
				},
				{
					"description": bson.M{
						"$regex": primitive.Regex{
							Pattern: s,
							Options: "i",
						},
					},
				},
			},
		}
	}

	if sort := c.Query("sort"); sort != "" {
		if sort == "asc" {
			findOptions.SetSort(bson.D{{"price", 1}})
		} else if sort == "desc" {
			findOptions.SetSort(bson.D{{"price", -1}})
		}
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	var perPage int64 = 9

	total, _ := collection.CountDocuments(ctx, filter)

	findOptions.SetSkip((int64(page) - 1) * perPage)
	findOptions.SetLimit(perPage)

	cursor, _ := collection.Find(ctx, filter, findOptions)
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var product Product
		cursor.Decode(&product)
		products = append(products, product)
	}

	return c.JSON(fiber.Map{
		"data":      products,
		"total":     total,
		"page":      page,
		"last_page": math.Ceil(float64(total / perPage)),
	})
}

func Insert(c *fiber.Ctx) error {
	db := InitMongoDB()

	collection := db.Collection("products")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	for i := 0; i < 50; i++ {
		collection.InsertOne(ctx, Product{
			Title:       faker.Word(),
			Description: faker.Paragraph(),
			Image:       fmt.Sprintf("http://lorempixel.com/200/200?%s", faker.UUIDDigit()),
			Price:       rand.Intn(90) + 10,
		})
	}

	return c.JSON(fiber.Map{
		"message": "success",
	})
}

func FindAll(c *fiber.Ctx) error {
	db := InitMongoDB()
	collection := db.Collection("products")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	products := []Product{}
	cursor, _ := collection.Find(ctx, bson.M{})
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var p Product
		cursor.Decode(&p)
		products = append(products, p)
	}
	return c.JSON(fiber.Map{"products": products})
}
