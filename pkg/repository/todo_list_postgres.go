package repository

import (
	"fmt"
	"github/todo-app"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type TodoListPostgres struct {
	db *sqlx.DB
}

func NewTodoListPostgres(db *sqlx.DB) *TodoListPostgres {
	return &TodoListPostgres{db: db}
}

func (r *TodoListPostgres) Create(userId int, list todo.TodoList) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	var id int
	createListQuery := fmt.Sprintf("INSERT INTO %s(title, description) VALUES ($1, $2) RETURNING id", todoListsTable)
	row := tx.QueryRow(createListQuery, list.Title, list.Description)
	if err := row.Scan(&id); err != nil {
		tx.Rollback()
		return 0, err
	}

	createUserListQuery := fmt.Sprintf("INSERT INTO %s(user_id, list_id) VALUES ($1, $2)", userListsTable)
	_, err = tx.Exec(createUserListQuery, userId, id)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit()
}

func (r *TodoListPostgres) GetAll(userId int) ([]todo.TodoList, error) {
	var lists []todo.TodoList
	//query := fmt.Sprintf("SELECT title, description FROM %s tl WHERE id IN (SELECT list_id FROM %s WHERE user_id = $1);", todoListsTable, userListsTable)
	query := fmt.Sprintf("SELECT tl.id, title, description FROM %s tl JOIN %s ul ON tl.id = ul.list_id WHERE ul.user_id = $1;",
		todoListsTable, userListsTable)
	err := r.db.Select(&lists, query, userId)
	return lists, err
}

func (r *TodoListPostgres) GetById(userId, listId int) (todo.TodoList, error) {
	list := todo.TodoList{Id: listId}
	query := fmt.Sprintf("SELECT tl.id, title, description FROM %s tl JOIN %s ul ON tl.id = ul.list_id WHERE ul.user_id = $1 AND tl.id = $2;",
		todoListsTable, userListsTable)
	err := r.db.Get(&list, query, userId, listId)
	return list, err
}

func (r *TodoListPostgres) Delete(userId, listId int) error {
	deleteQuery := fmt.Sprintf("DELETE FROM %s tl WHERE tl.id IN (SELECT ul.list_id FROM %s ul WHERE ul.user_id = $1 AND ul.list_id = $2)", todoListsTable, userListsTable)
	_, err := r.db.Exec(deleteQuery, userId, listId)
	return err
}

func (r *TodoListPostgres) Update(userId, listId int, input todo.UpdateListInput) error {
	setValues := make([]string, 0)
	args := make([]interface{}, 0)
	argId := 1

	if input.Title != nil {
		setValues = append(setValues, fmt.Sprintf("title=$%d", argId))
		args = append(args, *input.Title)
		argId++
	}

	if input.Description != nil {
		setValues = append(setValues, fmt.Sprintf("description=$%d", argId))
		args = append(args, *input.Description)
		argId++
	}
	setQuery := strings.Join(setValues, ",")
	query := fmt.Sprintf("UPDATE %s tl SET %s FROM %s ul WHERE tl.id = ul.list_id AND ul.list_id=$%d AND ul.user_id = $%d", todoListsTable, setQuery, userListsTable, argId, argId+1)

	//testing query row
	args = append(args, listId, userId)
	logrus.Debugf("updateQuery: %s", query)
	logrus.Debugf("args: %s", args)

	_, err := r.db.Exec(query, args...)
	return err
}
