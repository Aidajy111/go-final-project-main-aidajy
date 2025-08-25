package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const dateFormat = "20060102"

func NextDate(now time.Time, dstart string, repeat string) (string, error) {
	// Проверка на пустое правило
	if repeat == "" {
		return "", errors.New("empty repeat rule")
	}

	// Парсим исходную дату
	date, err := time.Parse(dateFormat, dstart)
	if err != nil {
		return "", errors.New("invalid date format")
	}

	// Разбиваем правило на части
	parts := strings.Fields(repeat)
	if len(parts) == 0 {
		return "", errors.New("invalid repeat format")
	}

	// Для повторяющихся задач начинаем с today, а не с dstart
	startDate := date
	if parts[0] == "d" || parts[0] == "y" {
		// Для повторяющихся задач используем today как начальную точку
		// если dstart в прошлом, или начинаем с today если dstart в будущем
		if date.Before(now) {
			startDate = now
		} else {
			startDate = now
		}
	}

	switch parts[0] {
	case "d":
		// Обработка дней
		if len(parts) != 2 {
			return "", errors.New("invalid d format")
		}

		days, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", errors.New("invalid days number")
		}
		if days <= 0 || days > 400 {
			return "", errors.New("days must be between 1 and 400")
		}

		result := startDate
		// если дата == сегодня или в будущем — оставляем её
		if !result.Before(now) {
			return result.Format(dateFormat), nil
		}

		// иначе ищем ближайшую будущую дату
		for {
			result = result.AddDate(0, 0, days)
			if !result.Before(now) {
				break
			}
		}
		return result.Format(dateFormat), nil

	case "y":
		// Обработка лет
		result := startDate
		for {
			result = result.AddDate(1, 0, 0)
			if result.After(now) {
				break
			}
		}
		return result.Format(dateFormat), nil

	default:
		// Неподдерживаемые правила
		return "", errors.New("unsupported repeat format")
	}
}

func NextDateHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	// Если now не указан, используем текущую дату
	var now time.Time
	if nowStr == "" {
		now = time.Now()
	} else {
		var err error
		now, err = time.Parse(dateFormat, nowStr)
		if err != nil {
			http.Error(w, "invalid now parameter", http.StatusBadRequest)
			return
		}
	}

	// Вычисляем следующую дату
	nextDate, err := NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Возвращаем результат
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(nextDate))
}
