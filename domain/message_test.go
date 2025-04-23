package domain

import (
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

var created_at = time.Now()

func TestMessageRepo_Get_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	createdAt := time.Now()
	rows := sqlmock.NewRows([]string{"Id", "Title", "Body", "CreatedAt"}).
		AddRow(1, "title", "body", createdAt)

	mock.ExpectPrepare("SELECT (.+) FROM messages").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(rows)

	got, err := repo.Get(1)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.EqualValues(t, 1, got.Id)
	assert.Equal(t, "title", got.Title)
	assert.Equal(t, "body", got.Body)
	assert.WithinDuration(t, createdAt, got.CreatedAt, time.Second)
}

func TestMessageRepo_Get_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	rows := sqlmock.NewRows([]string{"Id", "Title", "Body", "CreatedAt"})
	mock.ExpectPrepare("SELECT (.+) FROM messages").
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(rows)

	got, err := repo.Get(1)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Equal(t, "not_found", err.Error())
}

func TestMessageRepo_Get_InvalidPrepare(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	mock.ExpectPrepare("SELECT (.+) FROM wrong_table").
		ExpectQuery().
		WithArgs(1).
		WillReturnError(fmt.Errorf("prepare error"))

	got, err := repo.Get(1)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.Equal(t, "server_error", err.Error())
}

func TestMessageRepo_Create_OK(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	tm := time.Now()
	mock.ExpectPrepare("INSERT INTO messages").
		ExpectExec().
		WithArgs("title", "body", tm).
		WillReturnResult(sqlmock.NewResult(1, 1))

	input := &Message{
		Title:     "title",
		Body:      "body",
		CreatedAt: tm,
	}
	msg, err := repo.Create(input)

	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.EqualValues(t, 1, msg.Id)
	assert.Equal(t, "title", msg.Title)
	assert.Equal(t, "body", msg.Body)
	assert.WithinDuration(t, tm, msg.CreatedAt, time.Second)
}

func TestMessageRepo_Create_EmptyTitle(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	tm := time.Now()
	mock.ExpectPrepare("INSERT INTO messages").
		ExpectExec().
		WithArgs("title", "body", tm).
		WillReturnError(errors.New("empty title"))

	input := &Message{
		Title:     "",
		Body:      "body",
		CreatedAt: tm,
	}
	msg, err := repo.Create(input)
	assert.Nil(t, msg)
	assert.Error(t, err)
	assert.Equal(t, "server_error", err.Error())
}

func TestMessageRepo_Create_EmptyBody(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	tm := time.Now()
	mock.ExpectPrepare("INSERT INTO messages").
		ExpectExec().
		WithArgs("title", "body", tm).
		WillReturnError(errors.New("empty body"))

	input := &Message{
		Title:     "title",
		Body:      "",
		CreatedAt: tm,
	}
	msg, err := repo.Create(input)
	assert.Nil(t, msg)
	assert.Error(t, err)
	assert.Equal(t, "server_error", err.Error())
}

func TestMessageRepo_Create_InvalidSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()
	repo := NewMessageRepository(db)

	tm := time.Now()
	mock.ExpectPrepare("INSERT INTO wrong_table").
		ExpectExec().
		WithArgs("title", "body", tm).
		WillReturnError(errors.New("invalid sql query"))

	input := &Message{
		Title:     "title",
		Body:      "body",
		CreatedAt: tm,
	}
	msg, err := repo.Create(input)
	assert.Nil(t, msg)
	assert.Error(t, err)
	assert.Equal(t, "server_error", err.Error())
}

func TestUpdate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening stub db: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 1, Title: "update title", Body: "update body"}
	mock.ExpectPrepare("UPDATE messages").
		ExpectExec().WithArgs("update title", "update body", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := repo.Update(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, msg) {
		t.Errorf("expected: %v, got: %v", msg, got)
	}
}

func TestUpdate_InvalidSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 1, Title: "update title", Body: "update body"}
	mock.ExpectPrepare("UPDATER messages").
		ExpectExec().WithArgs("update title", "update body", 1).
		WillReturnError(errors.New("invalid SQL"))

	_, err = repo.Update(msg)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestUpdate_InvalidId(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 0, Title: "update title", Body: "update body"}
	mock.ExpectPrepare("UPDATE messages").
		ExpectExec().WithArgs("update title", "update body", 0).
		WillReturnError(errors.New("invalid update id"))

	_, err = repo.Update(msg)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestUpdate_EmptyTitle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 1, Title: "", Body: "update body"}
	mock.ExpectPrepare("UPDATE messages").
		ExpectExec().WithArgs("", "update body", 1).
		WillReturnError(errors.New("Please enter a valid title"))

	_, err = repo.Update(msg)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestUpdate_EmptyBody(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 1, Title: "update title", Body: ""}
	mock.ExpectPrepare("UPDATE messages").
		ExpectExec().WithArgs("update title", "", 1).
		WillReturnError(errors.New("Please enter a valid body"))

	_, err = repo.Update(msg)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestUpdate_UpdateFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	msg := &Message{Id: 1, Title: "update title", Body: "update body"}
	mock.ExpectPrepare("UPDATE messages").
		ExpectExec().WithArgs("update title", "update body", 1).
		WillReturnError(errors.New("Update failed"))

	_, err = repo.Update(msg)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestDelete_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening stub db: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	mock.ExpectPrepare("DELETE FROM messages").
		ExpectExec().WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	mock.ExpectPrepare("DELETE FROM messages").
		ExpectExec().WithArgs(100).
		WillReturnError(errors.New("Row not found"))

	err = repo.Delete(100)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestDelete_InvalidSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()
	repo := NewMessageRepository(db)

	mock.ExpectPrepare("DELETE FROMSSSS messages").
		ExpectExec().WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(1)
	if err == nil {
		t.Error("expected error but got none")
	}
}

func TestMessageRepo_Initialize(t *testing.T) {
	dbdriver := "mysql"
	username := "username"
	password := "password"
	host := "host"
	database := "database"
	port := "port"
	dbConnect := MessageRepo.Initialize(dbdriver, username, password, port, host, database)
	fmt.Println("this is the pool: ", dbConnect)
}
