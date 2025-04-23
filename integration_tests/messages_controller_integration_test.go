package integration_tests

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing-project/controllers"
	"testing-project/domain"
	"testing-project/utils/rabbitmq_utils"
)

func TestCreateMessage_Integration(t *testing.T) {
	domain.MessageRepo = &mockRepo{}

	rabbitmq_utils.PublishToQueue = func(message string) {
		t.Logf("Mock RabbitMQ called with message: %s", message)
	}

	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/messages", controllers.CreateMessage)

	payload := map[string]string{
		"title": "Integration Test",
		"body":  "This is a message",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/messages", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created, got %d", resp.Code)
	}

	var responseBody domain.Message
	if err := json.Unmarshal(resp.Body.Bytes(), &responseBody); err != nil {
		t.Errorf("Invalid response JSON: %s", err)
	}
	if responseBody.Title != "Integration Test" {
		t.Errorf("Expected title 'Integration Test', got '%s'", responseBody.Title)
	}
	if responseBody.Body != "This is a message" {
		t.Errorf("Expected body 'This is a message', got '%s'", responseBody.Title)
	}
}
