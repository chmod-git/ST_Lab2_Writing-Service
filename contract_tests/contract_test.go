package contract_tests

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
)

func createExpect(t *testing.T) *httpexpect.Expect {
	baseURL := "http://localhost:8080"

	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  baseURL,
		Reporter: httpexpect.NewRequireReporter(t),
		Client: &http.Client{
			Timeout: time.Second * 5,
		},
	})
}

func TestCreateMessage_Success(t *testing.T) {
	e := createExpect(t)

	unique := time.Now().UnixNano()
	title := fmt.Sprintf("Event title %d", unique)
	body := fmt.Sprintf("Event body %d", unique)

	obj := e.POST("/messages").
		WithJSON(map[string]interface{}{
			"title": title,
			"body":  body,
		}).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	id := int(obj.Value("id").Number().Raw())

	timeout := time.After(5 * time.Second)
	var receivedEvent string

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatalf("failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %s", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		"my_queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to register consumer: %s", err)
	}

waitLoop:
	for {
		select {
		case msg := <-msgs:
			receivedEvent = string(msg.Body)
			break waitLoop
		case <-timeout:
			t.Fatal("Timeout waiting for RabbitMQ message")
		}
	}

	var payload struct {
		Event string                 `json:"event"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(receivedEvent), &payload); err != nil {
		t.Fatalf("Invalid JSON from RabbitMQ: %s", err)
	}

	if payload.Event != "created" {
		t.Errorf("Expected event 'created', got %s", payload.Event)
	}
	if int(payload.Data["id"].(float64)) != id {
		t.Errorf("Expected id %d, got %v", id, payload.Data["id"])
	}
	if payload.Data["title"] != title {
		t.Errorf("Expected title %s, got %v", title, payload.Data["title"])
	}
	if payload.Data["body"] != body {
		t.Errorf("Expected body %s, got %v", body, payload.Data["body"])
	}
}

func TestCreateMessage_InvalidPayload(t *testing.T) {
	e := createExpect(t)

	e.POST("/messages").
		WithJSON(map[string]interface{}{
			"title": "",
			"body":  "",
		}).
		Expect().
		Status(http.StatusUnprocessableEntity).
		JSON().Object().
		Value("message").String().NotEmpty()
}

func TestUpdateMessage_Success(t *testing.T) {
	e := createExpect(t)

	unique := time.Now().UnixNano()
	origTitle := fmt.Sprintf("Original title %d", unique)
	origBody := fmt.Sprintf("Original body %d", unique)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatalf("failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %s", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		"my_queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to register consumer: %s", err)
	}

	msg := e.POST("/messages").
		WithJSON(map[string]interface{}{
			"title": origTitle,
			"body":  origBody,
		}).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	id := int(msg.Value("id").Number().Raw())

	updatedTitle := fmt.Sprintf("Updated title %d", unique)
	updatedBody := fmt.Sprintf("Updated body %d", unique)

	updated := e.PUT(fmt.Sprintf("/messages/%v", id)).
		WithJSON(map[string]interface{}{
			"title": updatedTitle,
			"body":  updatedBody,
		}).
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	updated.Value("title").String().Equal(updatedTitle)
	updated.Value("body").String().Equal(updatedBody)

	timeout := time.After(5 * time.Second)
	var createdReceived, updatedReceived bool

	for !(createdReceived && updatedReceived) {
		select {
		case msg := <-msgs:
			var payload struct {
				Event string                 `json:"event"`
				Data  map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				t.Fatalf("Invalid JSON from RabbitMQ: %s", err)
			}

			switch payload.Event {
			case "created":
				if int(payload.Data["id"].(float64)) != id {
					t.Errorf("Expected created id %d, got %v", id, payload.Data["id"])
				}
				if payload.Data["title"] != origTitle {
					t.Errorf("Expected created title %s, got %v", origTitle, payload.Data["title"])
				}
				if payload.Data["body"] != origBody {
					t.Errorf("Expected created body %s, got %v", origBody, payload.Data["body"])
				}
				createdReceived = true

			case "updated":
				if int(payload.Data["id"].(float64)) != id {
					t.Errorf("Expected updated id %d, got %v", id, payload.Data["id"])
				}
				if payload.Data["title"] != updatedTitle {
					t.Errorf("Expected updated title %s, got %v", updatedTitle, payload.Data["title"])
				}
				if payload.Data["body"] != updatedBody {
					t.Errorf("Expected updated body %s, got %v", updatedBody, payload.Data["body"])
				}
				updatedReceived = true
			}

		case <-timeout:
			if !createdReceived {
				t.Error("Did not receive 'created' event from RabbitMQ")
			}
			if !updatedReceived {
				t.Error("Did not receive 'updated' event from RabbitMQ")
			}
			t.Fatal("Timeout waiting for RabbitMQ messages")
		}
	}
}

func TestUpdateMessage_InvalidJson(t *testing.T) {
	e := createExpect(t)

	e.PUT("/messages/1").
		WithBytes([]byte(`invalid-json`)).
		Expect().
		Status(http.StatusUnprocessableEntity).
		JSON().Object().
		Value("message").String().Equal("invalid json body")
}

func TestDeleteMessage_Success(t *testing.T) {
	e := createExpect(t)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Fatalf("failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("failed to open channel: %s", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		"my_queue",
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to register consumer: %s", err)
	}

	msg := e.POST("/messages").
		WithJSON(map[string]interface{}{
			"title": "To be deleted",
			"body":  "Will be gone",
		}).
		Expect().
		Status(http.StatusCreated).
		JSON().Object()

	id := int(msg.Value("id").Number().Raw())

	e.DELETE(fmt.Sprintf("/messages/%v", id)).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("status").String().Equal("deleted")

	e.GET(fmt.Sprintf("/messages/%v", id)).
		Expect().
		Status(http.StatusNotFound)

	timeout := time.After(5 * time.Second)
	var createdReceived, deletedReceived bool

	for !(createdReceived && deletedReceived) {
		select {
		case msg := <-msgs:
			var payload struct {
				Event string                 `json:"event"`
				Data  map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				t.Fatalf("Invalid JSON from RabbitMQ: %s", err)
			}

			switch payload.Event {
			case "created":
				if int(payload.Data["id"].(float64)) == id {
					createdReceived = true
				}
			case "deleted":
				if int(payload.Data["id"].(float64)) == id {
					deletedReceived = true
				}
			}

		case <-timeout:
			if !createdReceived {
				t.Error("Did not receive 'created' event from RabbitMQ")
			}
			if !deletedReceived {
				t.Error("Did not receive 'deleted' event from RabbitMQ")
			}
			t.Fatal("Timeout waiting for RabbitMQ messages")
		}
	}
}

func TestDeleteMessage_NonExistent(t *testing.T) {
	e := createExpect(t)

	e.DELETE("/messages/999999").
		Expect().
		Status(http.StatusNotFound).
		JSON().Object().
		Value("message").String().Equal("no record matching given id")
}

func TestDeleteMessage_InvalidId(t *testing.T) {
	e := createExpect(t)

	e.DELETE("/messages/abc").
		Expect().
		Status(http.StatusBadRequest).
		JSON().Object().
		Value("message").String().Equal("message id should be a number")
}
