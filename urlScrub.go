// urlScrab
package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/opesun/goquery"
)

var opts struct {
	FileNameCompany string `short:"o" long:"open" default:"./names.txt" description:"With the names of the companies file"`
	FileFinal       string `short:"f" long:"final" default:"./final.csv" description:"The file with the saved information about the companies"`
}

func main() {
	flags.Parse(&opts)
	pwdDir, _ := os.Getwd()
	// создание файла log для записи ошибок
	fLog, err := os.OpenFile(pwdDir+`/log.txt`, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Fatalln(err)
	}
	defer fLog.Close()

	// создание файла отчета
	file, err := os.OpenFile(opts.FileFinal, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	// TODO: заголовок привести к нормальному виду после корректировки вывода
	/*
		getFile, err := file.Stat()
		if err != nil {
			log.Fatalln(err)
		}

		if getFile.Size() <= 1 {
			// заголовок
			file.WriteString("Наименование;ФИО директора;Положение директора;Виды деятельности;Дата регистрации;Кол-во сотрудников;ИНН;КПП;ОГРН;ОКПО;Адрес;Сайт;Место в категории;Уставной капитал;Основной заказчик\n")
		}
	*/
	// разобрать названия компаний для перебора
	var massName []string
	fileOpen, err := os.Open(opts.FileNameCompany)
	if err != nil {
		log.Fatalln(err)
	}
	// построчное считывание
	scanner := bufio.NewScanner(fileOpen)
	for scanner.Scan() {
		massName = append(massName, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	defer fileOpen.Close()

	for i := 0; i < len(massName); i++ {
		search := massName[i]

		// запрос по url
		resp, err := http.Get("https://www.google.ru/search?q=" + search + "+inurl%3Asbis.ru")
		if err != nil {
			log.Fatalln(err)
		}

		// отложенное закрытие коннекта
		defer resp.Body.Close()

		// парсинг ответа
		x, err := goquery.Parse(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		// храниение итоговых ссылок
		var urlsSearchs []string

		// формирование нормальной ссылки
		for _, l := range x.Find("h3").HtmlAll() {
			// обрезка html
			j := strings.TrimLeft(l, `<a href="/url?q=`)
			// надо убрать "левый" код в ссылке
			k := strings.Split(j, `&amp;sa=U&amp;ved=`)
			// итоговая ссылка готова
			urlsSearchs = append(urlsSearchs, "h"+k[0])
		}

		lenURL := 3
		if len(urlsSearchs) < 3 {
			lenURL = len(urlsSearchs)
		}
		for o := 0; o < lenURL; o++ {
			searchURL(urlsSearchs[o], file)
		}
	}
	log.Println("Готово")
}

func searchURL(url string, file *os.File) {
	x, err := goquery.ParseUrl(url)
	if err == nil {
		// обрезать от ненужной контактной информации
		urlData := strings.Split(x.Find(".content").Text(), "Оценка проведена на основании информации")
		// обработать для записи
		massData := strings.Split(urlData[0], "  ")
		for j := 0; j < len(massData)-1; j++ {
			massData[j] = strings.Trim(massData[j], " ")
		}

		// блок предварительной фильтрации
		var finalData []string
		for _, str := range massData {
			// уберем пустые строки
			if str != "" {
				finalData = append(finalData, str)
			}
		}

		// запись строки в файл (добавление)
		if len(finalData) > 1 {
			_, err := file.WriteString(url + "	")
			if err != nil {
				log.Fatalln(err)
			}
			for y, x := range finalData {
				// TODO: обработка вывода для записи
				// дальше разобрать через регулярки и номера подстрок
				switch {
				case y == 1:
					// короткое наименование компании
					writeString(x, file)
				case y == 2:
					// полное наименование компании
					writeString(x, file)
				case y == 3:
					// ФИО руководителя
					writeString(x, file)
				case y == 4:
					// положение директора
					writeString(strings.ToLower(x), file)
				case y == 5 || y == 6:
					reStr, _ := regexp.Compile(`еще`)
					// основная деятельность
					if !reStr.MatchString(x) {
						writeString(x, file)
					}
				case y == 7 || y == 8:
					reStr, _ := regexp.Compile(`\d\d\.\d\d\.\d\d\d\d`)
					// период регистрации компании
					if reStr.MatchString(x) {
						writeString(x, file)
					}
				case y > 9 && y < 14:
					reStr, _ := regexp.Compile(`\d{10}`)
					// ИНН
					if reStr.MatchString(x) {
						writeString(x, file)
					}
				}
			}
			_, err = file.WriteString("\n")
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	log.Printf("Ошибка парсинга страницы:\t%v\n", err)
}

// запись в файл строки
func writeString(x string, file *os.File) {
	// tab в виде разделителя
	_, err := file.WriteString(x + "	")
	if err != nil {
		log.Fatalln(err)
	}
}
