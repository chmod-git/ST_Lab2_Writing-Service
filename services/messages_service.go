package services

import (
	"encoding/json"
	"testing-project/domain"
	"testing-project/utils/error_utils"
	utils "testing-project/utils/rabbitmq_utils"
	"time"
)

var (
	MessagesService messageServiceInterface = &messagesService{}
)

type messagesService struct{}

type messageServiceInterface interface {
	GetMessage(int64) (*domain.Message, error_utils.MessageErr)
	CreateMessage(*domain.Message) (*domain.Message, error_utils.MessageErr)
	UpdateMessage(*domain.Message) (*domain.Message, error_utils.MessageErr)
	DeleteMessage(int64) error_utils.MessageErr
}

func (m *messagesService) GetMessage(msgId int64) (*domain.Message, error_utils.MessageErr) {
	return domain.MessageRepo.Get(msgId)
}

func (m *messagesService) CreateMessage(message *domain.Message) (*domain.Message, error_utils.MessageErr) {
	if err := message.Validate(); err != nil {
		return nil, err
	}
	message.CreatedAt = time.Now()
	message, err := domain.MessageRepo.Create(message)
	if err != nil {
		return nil, err
	}

	sendEvent("created", message)
	return message, nil
}

func (m *messagesService) UpdateMessage(message *domain.Message) (*domain.Message, error_utils.MessageErr) {
	if err := message.Validate(); err != nil {
		return nil, err
	}
	current, err := domain.MessageRepo.Get(message.Id)
	if err != nil {
		return nil, err
	}
	current.Title = message.Title
	current.Body = message.Body

	updated, err := domain.MessageRepo.Update(current)
	if err != nil {
		return nil, err
	}

	sendEvent("updated", updated)
	return updated, nil
}

func (m *messagesService) DeleteMessage(msgId int64) error_utils.MessageErr {
	msg, err := domain.MessageRepo.Get(msgId)
	if err != nil {
		return err
	}
	deleteErr := domain.MessageRepo.Delete(msg.Id)
	if deleteErr != nil {
		return deleteErr
	}

	sendEvent("deleted", msg)
	return nil
}

func sendEvent(eventType string, message *domain.Message) {
	event := map[string]interface{}{
		"event": eventType,
		"data":  message,
	}
	jsonMsg, err := json.Marshal(event)
	if err != nil {
		return
	}
	utils.PublishToQueue(string(jsonMsg))
}
