package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"testing-project/domain"
	"testing-project/utils/error_utils"
	utils "testing-project/utils/rabbitmq_utils"
	"time"
)

var (
	tm                   = time.Now()
	getMessageDomain     func(messageId int64) (*domain.Message, error_utils.MessageErr)
	createMessageDomain  func(msg *domain.Message) (*domain.Message, error_utils.MessageErr)
	updateMessageDomain  func(msg *domain.Message) (*domain.Message, error_utils.MessageErr)
	deleteMessageDomain  func(messageId int64) error_utils.MessageErr
	getAllMessagesDomain func() ([]domain.Message, error_utils.MessageErr)
)

type getDBMock struct{}

func (m *getDBMock) Get(messageId int64) (*domain.Message, error_utils.MessageErr) {
	return getMessageDomain(messageId)
}
func (m *getDBMock) Create(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
	return createMessageDomain(msg)
}
func (m *getDBMock) Update(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
	return updateMessageDomain(msg)
}
func (m *getDBMock) Delete(messageId int64) error_utils.MessageErr {
	return deleteMessageDomain(messageId)
}
func (m *getDBMock) GetAll() ([]domain.Message, error_utils.MessageErr) {
	return getAllMessagesDomain()
}
func (m *getDBMock) Initialize(string, string, string, string, string, string) *sql.DB {
	return nil
}

var publishedMessages []string

func mockPublishToQueue(msg string) {
	publishedMessages = append(publishedMessages, msg)
}

// "GetMessage" test cases

func TestMessagesService_GetMessage_Success(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:        1,
			Title:     "the title",
			Body:      "the body",
			CreatedAt: tm,
		}, nil
	}
	msg, err := MessagesService.GetMessage(1)
	fmt.Println("this is the message: ", msg)
	assert.NotNil(t, msg)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, msg.Id)
	assert.EqualValues(t, "the title", msg.Title)
	assert.EqualValues(t, "the body", msg.Body)
	assert.EqualValues(t, tm, msg.CreatedAt)
}

func TestMessagesService_GetMessage_NotFoundID(t *testing.T) {
	domain.MessageRepo = &getDBMock{}

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return nil, error_utils.NewNotFoundError("the id is not found")
	}
	msg, err := MessagesService.GetMessage(1)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, http.StatusNotFound, err.Status())
	assert.EqualValues(t, "the id is not found", err.Message())
	assert.EqualValues(t, "not_found", err.Error())
}

// Start of	"CreateMessage" test cases

func TestMessagesService_CreateMessage_Success(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	publishedMessages = nil
	utils.PublishToQueue = mockPublishToQueue

	createMessageDomain = func(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:        1,
			Title:     "the title",
			Body:      "the body",
			CreatedAt: tm,
		}, nil
	}

	request := &domain.Message{
		Title:     "the title",
		Body:      "the body",
		CreatedAt: tm,
	}
	msg, err := MessagesService.CreateMessage(request)

	assert.NotNil(t, msg)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, msg.Id)
	assert.EqualValues(t, "the title", msg.Title)
	assert.EqualValues(t, "the body", msg.Body)
	assert.Equal(t, 1, len(publishedMessages))

	var brokerMsg map[string]interface{}
	brokerErr := json.Unmarshal([]byte(publishedMessages[0]), &brokerMsg)
	assert.Nil(t, brokerErr)
	assert.Equal(t, "created", brokerMsg["event"])

	dataMap, ok := brokerMsg["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.EqualValues(t, 1, int64(dataMap["id"].(float64)))
	assert.Equal(t, "the title", dataMap["title"])
	assert.Equal(t, "the body", dataMap["body"])
}

func TestMessagesService_CreateMessage_InvalidTitle(t *testing.T) {
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	tm := time.Now()
	request := &domain.Message{
		Title:     "",
		Body:      "the body",
		CreatedAt: tm,
	}

	msg, err := MessagesService.CreateMessage(request)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, "Please enter a valid title", err.Message())
	assert.EqualValues(t, http.StatusUnprocessableEntity, err.Status())
	assert.EqualValues(t, "invalid_request", err.Error())
	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_CreateMessage_InvalidBody(t *testing.T) {
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	tm := time.Now()
	request := &domain.Message{
		Title:     "the title",
		Body:      "",
		CreatedAt: tm,
	}

	msg, err := MessagesService.CreateMessage(request)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, "Please enter a valid body", err.Message())
	assert.EqualValues(t, http.StatusUnprocessableEntity, err.Status())
	assert.EqualValues(t, "invalid_request", err.Error())
	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_CreateMessage_Failure(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	publishedMessages = nil
	utils.PublishToQueue = mockPublishToQueue

	createMessageDomain = func(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
		return nil, error_utils.NewInternalServerError("title already taken")
	}
	request := &domain.Message{
		Title:     "the title",
		Body:      "the body",
		CreatedAt: tm,
	}
	msg, err := MessagesService.CreateMessage(request)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, "title already taken", err.Message())
	assert.EqualValues(t, http.StatusInternalServerError, err.Status())
	assert.EqualValues(t, "server_error", err.Error())
	assert.Equal(t, 0, len(publishedMessages))
}

// "UpdateMessage" test cases

func TestMessagesService_UpdateMessage_Success(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	publishedMessages = nil
	utils.PublishToQueue = mockPublishToQueue

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:    1,
			Title: "former title",
			Body:  "former body",
		}, nil
	}
	updateMessageDomain = func(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:    1,
			Title: "the title update",
			Body:  "the body update",
		}, nil
	}
	request := &domain.Message{
		Title: "the title update",
		Body:  "the body update",
	}
	msg, err := MessagesService.UpdateMessage(request)

	assert.NotNil(t, msg)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, msg.Id)
	assert.EqualValues(t, "the title update", msg.Title)
	assert.EqualValues(t, "the body update", msg.Body)

	assert.Equal(t, 1, len(publishedMessages))

	var brokerMsg map[string]interface{}
	brokerErr := json.Unmarshal([]byte(publishedMessages[0]), &brokerMsg)
	assert.Nil(t, brokerErr)
	assert.Equal(t, "updated", brokerMsg["event"])

	dataMap, ok := brokerMsg["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.EqualValues(t, 1, int64(dataMap["id"].(float64)))
	assert.Equal(t, "the title update", dataMap["title"])
	assert.Equal(t, "the body update", dataMap["body"])
}

func TestMessagesService_UpdateMessage_EmptyTitle(t *testing.T) {
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	request := &domain.Message{
		Title: "",
		Body:  "the body",
	}

	msg, err := MessagesService.UpdateMessage(request)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, http.StatusUnprocessableEntity, err.Status())
	assert.EqualValues(t, "Please enter a valid title", err.Message())
	assert.EqualValues(t, "invalid_request", err.Error())
	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_UpdateMessage_EmptyBody(t *testing.T) {
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	request := &domain.Message{
		Title: "the title",
		Body:  "",
	}

	msg, err := MessagesService.UpdateMessage(request)
	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, http.StatusUnprocessableEntity, err.Status())
	assert.EqualValues(t, "Please enter a valid body", err.Message())
	assert.EqualValues(t, "invalid_request", err.Error())
	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_UpdateMessage_Failure_Getting_Former_Message(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return nil, error_utils.NewInternalServerError("error getting message")
	}
	request := &domain.Message{
		Title: "the title update",
		Body:  "the body update",
	}
	msg, err := MessagesService.UpdateMessage(request)

	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, "error getting message", err.Message())
	assert.EqualValues(t, http.StatusInternalServerError, err.Status())
	assert.EqualValues(t, "server_error", err.Error())

	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_UpdateMessage_Failure_Updating_Message(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:    1,
			Title: "former title",
			Body:  "former body",
		}, nil
	}
	updateMessageDomain = func(msg *domain.Message) (*domain.Message, error_utils.MessageErr) {
		return nil, error_utils.NewInternalServerError("error updating message")
	}
	request := &domain.Message{
		Title: "the title update",
		Body:  "the body update",
	}
	msg, err := MessagesService.UpdateMessage(request)

	assert.Nil(t, msg)
	assert.NotNil(t, err)
	assert.EqualValues(t, "error updating message", err.Message())
	assert.EqualValues(t, http.StatusInternalServerError, err.Status())
	assert.EqualValues(t, "server_error", err.Error())

	assert.Equal(t, 0, len(publishedMessages))
}

// Start of "DeleteMessage" test cases

func TestMessagesService_DeleteMessage_Success(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:    1,
			Title: "former title",
			Body:  "former body",
		}, nil
	}
	deleteMessageDomain = func(messageId int64) error_utils.MessageErr {
		return nil
	}

	err := MessagesService.DeleteMessage(1)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(publishedMessages))

	var brokerMsg map[string]interface{}
	brokerErr := json.Unmarshal([]byte(publishedMessages[0]), &brokerMsg)
	assert.Nil(t, brokerErr)
	assert.Equal(t, "deleted", brokerMsg["event"])

	dataMap, ok := brokerMsg["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.EqualValues(t, 1, int64(dataMap["id"].(float64)))
	assert.Equal(t, "former title", dataMap["title"])
	assert.Equal(t, "former body", dataMap["body"])
}

func TestMessagesService_DeleteMessage_Error_Getting_Message(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return nil, error_utils.NewInternalServerError("Something went wrong getting message")
	}

	err := MessagesService.DeleteMessage(1)
	assert.NotNil(t, err)
	assert.EqualValues(t, "Something went wrong getting message", err.Message())
	assert.EqualValues(t, http.StatusInternalServerError, err.Status())
	assert.EqualValues(t, "server_error", err.Error())

	assert.Equal(t, 0, len(publishedMessages))
}

func TestMessagesService_DeleteMessage_Error_Deleting_Message(t *testing.T) {
	domain.MessageRepo = &getDBMock{}
	utils.PublishToQueue = mockPublishToQueue
	publishedMessages = nil

	getMessageDomain = func(messageId int64) (*domain.Message, error_utils.MessageErr) {
		return &domain.Message{
			Id:    1,
			Title: "former title",
			Body:  "former body",
		}, nil
	}
	deleteMessageDomain = func(messageId int64) error_utils.MessageErr {
		return error_utils.NewInternalServerError("error deleting message")
	}

	err := MessagesService.DeleteMessage(1)
	assert.NotNil(t, err)
	assert.EqualValues(t, "error deleting message", err.Message())
	assert.EqualValues(t, http.StatusInternalServerError, err.Status())
	assert.EqualValues(t, "server_error", err.Error())

	assert.Equal(t, 0, len(publishedMessages))
}
