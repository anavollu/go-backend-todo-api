package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Todo struct {
	gorm.Model
	Name string `json:"name"`
	Done bool   `json:"done"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	db, err := gorm.Open(postgres.Open(os.Getenv("DB_URL")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// TODO: change the migration method
	db.AutoMigrate(&Todo{})

	app := fiber.New()

	app.Use(logger.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOW_ORIGINS"),
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	todo := app.Group("/todo")

	todo.Get("/", func(c *fiber.Ctx) error {
		var todos []Todo
		page, err := strconv.Atoi(c.Query("page", "1"))
		if err != nil || page <= 0 {
			page = 1
		}
		limit, err := strconv.Atoi(c.Query("limit", "10"))
		if err != nil {
			limit = 10
		}
		result := db.Offset((page - 1) * limit).Limit(limit).Find(&todos)
		if result.Error != nil {
			return c.Status(http.StatusInternalServerError).SendString(result.Error.Error())
		}
		return c.JSON(struct {
			Page  int    `json:"page"`
			Limit int    `json:"limit"`
			Todos []Todo `json:"todos"`
		}{
			Page:  page,
			Limit: limit,
			Todos: todos,
		})
	})

	todo.Post("/", func(c *fiber.Ctx) error {
		todo := Todo{}
		err := c.BodyParser(&todo)
		if err != nil {
			return c.Status(http.StatusUnprocessableEntity).SendString(err.Error())
		}
		result := db.Create(&todo)
		if result.Error != nil {
			return c.Status(http.StatusInternalServerError).SendString(result.Error.Error())
		}
		return c.JSON(todo)
	})

	todo.Put("/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		todo := Todo{}
		err := c.BodyParser(&todo)
		if err != nil {
			return c.Status(http.StatusUnprocessableEntity).SendString(err.Error())
		}

		result := db.Model(&todo).Where("id = ?", id).Updates(&todo).Find(&todo)
		if result.Error != nil {
			return c.Status(http.StatusInternalServerError).SendString(result.Error.Error())
		}

		return c.JSON(todo)
	})

	todo.Delete("/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		todo := Todo{}

		db.Where("id = ?", id).Find(&todo)
		if todo.ID == 0 {
			return c.SendStatus(http.StatusAccepted)
		}

		result := db.Delete(&todo, id)
		if result.Error != nil {
			return c.Status(http.StatusInternalServerError).SendString(result.Error.Error())
		}

		return c.JSON(struct {
			ID        string    `json:"id"`
			DeletedAt time.Time `json:"deletedAt"`
		}{
			ID:        id,
			DeletedAt: todo.DeletedAt.Time,
		})
	})

	app.Listen(":3000")
}

// modular o sistema (opcional)
