package integration_tests

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"testing-project/domain"
	"testing-project/services"
	"testing-project/utils/error_utils"
	"testing-project/utils/rabbitmq_utils"
)

type mockRepo struct{}

func (m *mockRepo) Get(id int64) (*domain.Message, error_utils.MessageErr) {
	return nil, nil
}
func (m *mockRepo) Create(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
	msg.Id = 999
	return msg, nil
}
func (m *mockRepo) Update(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
	return nil, nil
}
func (m *mockRepo) Delete(id int64) error_utils.MessageErr {
	return nil
}
func (m *mockRepo) Initialize(_, _, _, _, _, _ string) *sql.DB {
	return nil
}

func TestCreateMessage_PublishesToRabbitMQ(t *testing.T) {
	domain.MessageRepo = &mockRepo{}

	called := false
	var published string

	rabbitmq_utils.PublishToQueue = func(message string) {
		called = true
		published = message
	}

	msg := &domain.Message{
		Title: "Test Publish",
		Body:  "This is the body",
	}

	created, err := services.MessagesService.CreateMessage(msg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !called {
		t.Errorf("Expected PublishToQueue to be called, but it wasn't")
	}

	if created.Id != 999 {
		t.Errorf("Expected ID to be set by repo, got %d", created.Id)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(published), &payload); err != nil {
		t.Errorf("Invalid JSON published: %v", err)
	}

	if payload["event"] != "created" {
		t.Errorf("Expected event type 'created', got %v", payload["event"])
	}

	data := payload["data"].(map[string]interface{})
	if !strings.Contains(data["title"].(string), "Test Publish") {
		t.Errorf("Incorrect title in published data: %v", data["title"])
	}
	if !strings.Contains(data["body"].(string), "This is the body") {
		t.Errorf("Incorrect body in published data: %v", data["title"])
	}
}
