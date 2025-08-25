package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Aidajy111/go-final-project-main/pkg/db"
)

func Init() {
	http.HandleFunc("/api/nextdate", NextDateHandler)
	http.HandleFunc("/api/task", taskHandler)
	http.HandleFunc("/api/tasks", GetTasks)
	http.HandleFunc("/api/task/done", doneHandler)
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		addTaskHandler(w, r)
	case http.MethodGet:
		getTaskHandler(w, r)
	case http.MethodPut:
		updateTaskHandler(w, r)
	case http.MethodDelete:
		deleteTaskHandler(w, r) // добавляем DELETE
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func GetTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasksHandler(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, taskResponse{Error: "method not allowed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error, status int) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

// Обработчик для GET /api/task?id=<id>
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	task, err := db.GetTask(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Обработчик для PUT /api/task
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var task db.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат данных"})
		return
	}

	// Валидация названия задачи
	if task.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Название задачи не может быть пустым"})
		return
	}

	// Валидация даты (должна быть в формате YYYYMMDD и быть корректной датой)
	if task.Date != "" {
		if len(task.Date) != 8 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат даты"})
			return
		}

		// Проверка что дата корректна (аналогично NextDate)
		_, err := time.Parse("20060102", task.Date)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат даты"})
			return
		}
	}

	// Валидация правила повторения (если указано)
	if task.Repeat != "" {
		// Проверяем что правило повторения валидно
		if !isValidRepeatRule(task.Repeat) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Неверный формат правила повторения"})
			return
		}
	}

	// Обновляем задачу в базе данных
	err = db.UpdateTask(&task)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	// Возвращаем пустой JSON при успешном обновлении
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}
func isValidRepeatRule(repeat string) bool {
	if repeat == "" {
		return true
	}

	// Проверяем базовые форматы: d X, y X, w X, m X
	if len(repeat) < 2 {
		return false
	}

	// Простая проверка - более сложная логика должна быть в NextDate
	switch repeat[0] {
	case 'd', 'y', 'w', 'm':
		// Должен быть пробел и число после него
		if len(repeat) < 3 || repeat[1] != ' ' {
			return false
		}
		// Проверяем что после пробела число
		for _, char := range repeat[2:] {
			if char < '0' || char > '9' {
				return false
			}
		}
		return true
	default:
		return false
	}
}
func doneHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	// Получаем задачу из базы
	task, err := db.GetTask(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	// Парсим дату задачи
	taskDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "invalid date in task"})
		return
	}

	now := time.Now()

	if task.Repeat == "" {
		// Одноразовая задача → удаляем
		if err := db.DeleteTask(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	} else {
		// Повторяющаяся задача
		var next string
		if !taskDate.Before(now) {
			// Задача ещё не выполнена сегодня — оставляем дату
			next = task.Date
		} else {
			// Задача в прошлом — вычисляем следующую дату
			next, err = NextDate(now, task.Date, task.Repeat)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}

		// Обновляем дату в базе
		if err := db.UpdateDate(next, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	// Возвращаем пустой JSON при успехе
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Не указан идентификатор"})
		return
	}

	if err := db.DeleteTask(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{})
}
