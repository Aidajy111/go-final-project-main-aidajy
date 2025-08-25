package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aidajy111/go-final-project-main/pkg/db"
)

type taskResponse struct {
	ID    int64  `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			writeJSON(w, http.StatusInternalServerError, taskResponse{Error: "internal server error"})
		}
	}()

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, taskResponse{Error: "method not allowed"})
		return
	}

	var task db.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSON(w, http.StatusBadRequest, taskResponse{Error: "invalid JSON format"})
		return
	}

	if task.Title == "" {
		writeJSON(w, http.StatusBadRequest, taskResponse{Error: "title is required"})
		return
	}

	now := time.Now()
	currentDate := now.Format("20060102")

	// Если дата не указана, используем сегодняшнюю
	if task.Date == "" {
		task.Date = currentDate
	}

	// Парсим указанную дату
	taskTime, err := time.Parse("20060102", task.Date)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, taskResponse{Error: "invalid date format"})
		return
	}

	// Обрабатываем повторяющиеся задачи
	if task.Repeat != "" {
		// Проверяем валидность правила повторения
		_, err := NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, taskResponse{Error: err.Error()})
			return
		}

		// Для ВСЕХ повторяющихся задач игнорируем указанную дату
		// и начинаем с сегодняшнего дня или следующей валидной даты
		nextDate, err := NextDate(now, currentDate, task.Repeat)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, taskResponse{Error: err.Error()})
			return
		}
		task.Date = nextDate
	} else {
		// Для разовых задач: если дата в прошлом, используем сегодняшнюю
		if taskTime.Before(now) {
			task.Date = currentDate
		}
		// Для разовых задач с будущей датой оставляем как есть
	}

	id, err := db.AddTask(&task)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, taskResponse{Error: "database error"})
		return
	}

	writeJSON(w, http.StatusOK, taskResponse{ID: id})
}
