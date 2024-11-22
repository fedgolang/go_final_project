package taskrepeat

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
	if len(repeatSlice) > 1 {
		value = repeatSlice[1]
	}
	switch rule {
	case "d":
		// Пробуем перевести число дней в int
		valueInt, err := strconv.Atoi(value)
		// Если нам передали непереводимое в число значение, выходим
		if err != nil {
			return "", err
		}

		// По ТЗ, если значение больше чем 400, выходим
		if valueInt > 400 {
			return "", fmt.Errorf("превышен максимально допустимый интервал")
		}

		// Необходимо отдельное условие для событий
		// Когда date уже больше чем now
		if dateParse.After(now) {
			nextDate = fmt.Sprint(dateParse.AddDate(0, 0, valueInt))
			return nextDate, nil
		}

		// Запустим цикл, который будет добавлять к дате наш интервал,
		// пока дата не перерастёт now
		for {
			// Сразу добавляем интервал
			dateParse = dateParse.AddDate(0, 0, valueInt)
			// Если старт позже чем now - выходим
			if dateParse.After(now) {
				break
			}
		}

		// Следующая дата и будет нашей искомой
		nextDate = fmt.Sprint(dateParse)
	case "y":
		// Необходимо отдельное условие для событий
		// Когда date уже больше чем now
		if dateParse.After(now) {
			nextDate = fmt.Sprint(dateParse.AddDate(1, 0, 0))
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

		// Следующая дата и будет нашей искомой
		nextDate = fmt.Sprint(dateParse)
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

		// Пройдем по слайсу, сколько бы значений не было, начинаем с завтра
		// Создал итератор так как к i привязываться нельзя из-за старта today + 1
		j := 1
		for i := today + 1; i < 8; i++ {
			if search(i, weekInts) {
				nextDate = fmt.Sprint(dateParse.AddDate(0, 0, j))
				break
			}
			if i == 7 {
				i = 1
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
		monthsOfYear := []string{}
		if len(repeatSlice) > 2 {
			month = repeatSlice[2]
		}
		if month != "" {
			monthsOfYear = strings.Split(month, ",")
		}

		// Берём все дни из обязательной последовательности
		daysOfMonth := strings.Split(value, ",")

		// Переведём строковые значения дней в числа
		intDaysOfMonth := []int{}
		for _, day := range daysOfMonth {
			intDay, err := strconv.Atoi(day)
			if err != nil {
				return "", err
			}
			intDaysOfMonth = append(intDaysOfMonth, intDay)
		}

		// Пройдёмся по всем числам внутри intDaysOfMonth
		target := now.Day() + 1
		for {
			if search(target, intDaysOfMonth) {
				nextDate = fmt.Sprint(now.AddDate(0, 0, target))
				break
			}
			target++
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

func lastDayOfMonth(year int, month time.Month) time.Time {
	// Ищем первый день следующего месяца
	firstDayNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	// Вычитаем один день
	lastDay := firstDayNextMonth.AddDate(0, 0, -1)
	return lastDay
}
