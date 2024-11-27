package nextdate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	var nextDate string
	var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

	// Проверим, не передали ли нам пустую строку.
	// Так же уберём из неё все символы не алфавитного типа.
	if nonAlphanumericRegex.ReplaceAllString(strings.TrimSpace(repeat), "") == "" {
		return "", fmt.Errorf("обнаружена некорректная строка в атрибуте repeat")
	}

	// Если у нас пустое значение repeat, будем удалять таску
	if repeat == "" {
		return "mark for delete", nil
	}

	// Проверяем, дали ли нам корректную дату
	dateParse, err := time.Parse("20060102", date)
	if err != nil {
		return "", err
	}

	repeatSlice := strings.Split(repeat, " ")

	rule := repeatSlice[0]
	value := ""

	// Проверим, что нам пришли данные о днях или месяцах
	if len(repeatSlice) > 1 {
		// И проверим, что они не пустые
		if len(strings.TrimSpace(repeatSlice[1])) > 0 {
			value = repeatSlice[1]
		}
	}

	switch rule {
	case "d":
		// Пробуем перевести число дней в int
		valueInt, err := strconv.Atoi(value)
		// Если нам передали непереводимое в число значение, выходим
		if err != nil {
			return "", fmt.Errorf("значение дня передано некорректно: %s", err)
		}

		// По ТЗ, если значение больше чем 400, выходим
		if valueInt > 400 {
			return "", fmt.Errorf("превышен максимально допустимый интервал количества дней")
		}

		// Проверим на каждодневность
		// Если дата < чем сейчас, перенесём на сегодня
		if valueInt == 1 && dateParse.Before(now) {
			return fmt.Sprint(now.Format("20060102")), nil
		} else if valueInt == 1 && dateParse.After(now) {
			// Если > чем сейчас, перенос на следующий день
			return fmt.Sprint(dateParse.AddDate(0, 0, valueInt).Format("20060102")), nil
		}

		// Необходимо отдельное условие для событий
		// Когда date уже больше чем now
		if dateParse.After(now) {
			nextDate = fmt.Sprint(dateParse.AddDate(0, 0, valueInt).Format("20060102"))
			return nextDate, nil
		}

		// Запустим цикл, который будет добавлять к дате наш интервал
		for {
			if dateParse.After(now) {
				break
			}
			// Добавляем интервал
			dateParse = dateParse.AddDate(0, 0, valueInt)
		}

		nextDate = dateParse.Format("20060102")

	case "y":
		// Необходимо отдельное условие для событий
		// Когда date уже больше чем now
		if dateParse.After(now) {
			nextDate = fmt.Sprint(dateParse.AddDate(1, 0, 0).Format("20060102"))
			return nextDate, nil
		}

		// Запустим цикл, который будет добавлять к дате наш интервал,
		// пока дата не перерастёт now
		for {
			// Сразу добавляем интервал
			dateParse = dateParse.AddDate(1, 0, 0)
			// Если старт позже чем now - выходим
			if dateParse.After(now) {
				break
			}
		}

		nextDate = dateParse.Format("20060102")

	case "w":
		// Обработаем сразу ошибку пустого value
		if value == "" {
			return "", fmt.Errorf("при передаче правила %s, пришел пустой день недели", rule)
		}
		// Берём числовое значение текущего дня недели
		today := int(now.Weekday())
		// Для начала поменяем последовательность, у нас начало с понедельника
		if today == 0 {
			today = 7
		}

		// Отдельно сделаем проверку на завтрашний день
		// Чтобы не упасть на today + 1 в воскресенье
		var tomorrow int
		if today == 7 {
			tomorrow = 1
		} else {
			tomorrow = today + 1
		}

		// Разобьем значение value, в случае, если там более 1 значения
		week := strings.Split(value, ",")

		// Создадим отдельный слайс куда перекинем все значения week в int
		weekInts := []int{}
		for _, day := range week {
			dayInt, err := strconv.Atoi(day)
			if err != nil {
				return "", err
			}
			weekInts = append(weekInts, dayInt)
		}

		// Пройдём по слайсу weekInts чтобы проверить, что у нас нет числа меньше 1 и больше 7
		for _, day := range weekInts {
			if day < 1 || day > 7 {
				return "", fmt.Errorf("при передаче правила %s, пришел некорректный номер дня недели: %d", rule, day)
			}
		}

		// Узнаем откуда нам производить отсчёт
		var dateStart time.Time
		if dateParse.After(now) {
			dateStart = dateParse
		} else {
			dateStart = now
		}

		// Пройдем по слайсу, сколько бы значений не было, начинаем с завтра
		// Создал итератор так как к i привязываться нельзя из-за старта today + 1
		j := 1
		for i := tomorrow; i < 8; i++ {
			if search(i, weekInts) {
				nextDate = fmt.Sprint(dateStart.AddDate(0, 0, j).Format("20060102"))
				break
			}
			if i == 7 {
				i = 0
			}
			j++
		}
	case "m":
		// Проверяем что передана обязательная последовательность
		if value == "" {
			return "", fmt.Errorf("при передаче правила %s, пришел пустой день месяца", rule)
		}

		// Добавим переменные для проверки, есть ли необязательная последовательность
		month := ""
		intMonths := []int{}
		if len(repeatSlice) > 2 {
			month = repeatSlice[2]
		}
		if month != "" {
			monthsOfYear := strings.Split(month, ",")

			// Переводим месяца в int
			for _, month := range monthsOfYear {
				intMonth, err := strconv.Atoi(month)
				if err != nil {
					return "", err
				}
				if intMonth > 12 || intMonth < 1 {
					return "", fmt.Errorf("числовое значение месяца некорректно: %d", intMonth)
				}
				intMonths = append(intMonths, intMonth)
			}
		}

		// Берём все дни из обязательной последовательности
		daysOfMonth := strings.Split(value, ",")

		// Переведём строковые значения дней в числа
		intDaysOfMonth := []int{}
		for _, day := range daysOfMonth {
			intDay, err := strconv.Atoi(day)
			if err != nil || intDay > 31 || intDay < -2 {
				return "", err
			}

			intDaysOfMonth = append(intDaysOfMonth, intDay)
		}

		// Узнаем откуда нам производить отсчёт
		var dateStart time.Time
		if dateParse.After(now) {
			dateStart = dateParse
		} else {
			dateStart = now
		}
		nextDate = findNextDate(dateStart, intDaysOfMonth, intMonths).Format("20060102")
		// Если не нашлось кандидатов и нам вернулся дефолтный time.Time, значит ошибка
		if nextDate == "00010101" {
			return "", nil
		}
	default:
		return "", fmt.Errorf("неверный формат repeat")
	}

	return nextDate, nil
}

// Функция для сверки есть ли в слайсе искомое
func search(target int, slice []int) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// findNextDate находит ближайшую дату на основе правил.
func findNextDate(now time.Time, days, months []int) time.Time {
	currentYear, currentMonth := now.Year(), int(now.Month())
	var candidates []time.Time

	// Определяем, какие месяцы нужно учитывать
	monthSet := map[int]bool{}
	if len(months) == 0 {
		// Если месяцы не указаны, берем все 12
		for i := 1; i <= 12; i++ {
			monthSet[i] = true
		}
	} else {
		for _, m := range months {
			monthSet[m] = true
		}
	}

	// Генерируем даты
	for monthOffset := 0; monthOffset < 24; monthOffset++ { // 2 года вперёд для поиска
		month := (currentMonth+monthOffset-1)%12 + 1
		year := currentYear + (currentMonth+monthOffset-1)/12

		// Пропускаем месяцы, не указанные в правилах
		if !monthSet[month] {
			continue
		}

		// Для каждого указанного дня месяца добавляем кандидатов
		for _, day := range days {
			date := calculateDate(year, month, day)
			if date.After(now) {
				candidates = append(candidates, date)
			}
		}
	}

	// Найти минимальную дату
	if len(candidates) == 0 {
		return time.Time{}
	}
	minDate := candidates[0]
	for _, d := range candidates {
		if d.Before(minDate) {
			minDate = d
		}
	}
	return minDate
}

// calculateDate вычисляет дату на основе года, месяца и дня (включая -1 и -2).
func calculateDate(year, month, day int) time.Time {
	// Определяем количество дней в месяце
	lastDay := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()
	var targetDay int
	if day > 0 {
		targetDay = day
	} else if day == -1 {
		targetDay = lastDay
	} else if day == -2 {
		targetDay = lastDay - 1
	}

	// Проверяем, что день в допустимых пределах
	if targetDay < 1 || targetDay > lastDay {
		return time.Time{} // Возвращаем "нулевую" дату, если недопустимо
	}

	// Возвращаем рассчитанную дату
	return time.Date(year, time.Month(month), targetDay, 0, 0, 0, 0, time.Local)
}
