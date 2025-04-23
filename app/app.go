package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
	"testing-project/domain"
	"testing-project/utils/rabbitmq_utils"
)

var (
	router = gin.Default()
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Print("sad .env file found")
	}
}

func StartApp() {
	dbdriver := os.Getenv("DBDRIVER")
	username := os.Getenv("DB_USERNAME")
	password := os.Getenv("PASSWORD")
	host := os.Getenv("HOST")
	database := os.Getenv("DATABASE")
	port := os.Getenv("PORT")
	brokerAddr := os.Getenv("RABBITMQ_URL")

	domain.MessageRepo.Initialize(dbdriver, username, password, port, host, database)
	fmt.Println("DATABASE STARTED")

	rabbitmq_utils.InitRabbitMQ(brokerAddr)

	routes()

	router.Run(":8080")
}
