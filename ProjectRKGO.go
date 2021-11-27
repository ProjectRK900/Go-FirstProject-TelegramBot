// ======================= Hello World =======================
/*package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, playground")
}*/
// ===========================================================

// ========================= Web-test ========================
/*package main

import (
	"fmt"
	"github.com/goombaio/namegenerator"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", help)
	http.ListenAndServe("localhost:8067", nil)
}

func help(writer http.ResponseWriter, request *http.Request) {

	seed := time.Now().UTC().Unix()
	nameG := namegenerator.NewNameGenerator(seed)
	fmt.Fprintf(writer, "Hello %s", nameG.Generate())

}*/
// ===========================================================

// ======================= Telegram Bot ======================
package main

import ( // Библиотеки
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type binanceResp struct { // структура типа для данных, полученных с API
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}
type wallet map[string]float64 // тип "кошелёк" для ID чата

var db = map[int64]wallet{} // "БД"

func main() { // Главная функция
	bot, err := tgbotapi.NewBotAPI("2122826174:AAFjV_ISH_sdE2Oi0hWwFUpTeuuNbV8kugw") // API бота
	if err != nil {                                                                  // обработка ошибки
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName) // сообщение об успешной авторизации бота

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates { // обработка ботом сообщения пользователя
		if update.Message == nil { // ignore any non-Message Updates
			continue // обработка ошибки
		}

		log.Println(update.Message.Text)
		update.Message.Text += " easteregg"

		words := strings.Split(update.Message.Text, " ") // разбивка строки сообщения на слова
		switch words[0] {                                // конструкция switch/case, аналогична C# и C++
		case "+":
			sum, err := strconv.ParseFloat(words[2], 64) // конвертация строки в число
			if err != nil {                              // обработка ошибки
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка в преобразовании числа"))
				continue
			}

			if isExist(words[1]) {
				if _, ok := db[update.Message.Chat.ID]; !ok { // проверка, существует ли "кошелёк"
					db[update.Message.Chat.ID] = wallet{}
				}
				db[update.Message.Chat.ID][words[1]] += sum // добавление в кошелёк

				//msg := fmt.Sprintf("Баланс %c %f", words[1], db[update.Message.Chat.ID][words[1]])
				msg := "Баланс " + words[1] + ": " + fmt.Sprint(db[update.Message.Chat.ID][words[1]]) // сообщение-ответ бота
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))                            // отправка сообщения ботом
			} else {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка c валютой"))
				continue
			}

		case "-": // аналогично "+"
			minus, err := strconv.ParseFloat(words[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка в преобразовании числа"))
				continue
			}

			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}

			if (db[update.Message.Chat.ID][words[1]] - minus) < 0 { // проверка на вычитание
				db[update.Message.Chat.ID][words[1]] = 0 // большего значения, чем
			} else { // есть в "кошельке"
				db[update.Message.Chat.ID][words[1]] -= minus
			}

			msg := "Баланс " + words[1] + ": " + fmt.Sprint(db[update.Message.Chat.ID][words[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "delete": // удаление валюты
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}

			var msg string
			ex := false
			for key := range db[update.Message.Chat.ID] { // цикл проверки на наличие удаляемой
				if key == words[1] { // валюты
					delete(db[update.Message.Chat.ID], words[1]) // удаление с слайса
					msg = "Валюта " + words[1] + " удалена"
					ex = true
					break
				}
			}
			if !ex {
				msg = "Такой валюты нет в кошельке"
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		case "show": // вывод всего "кошелька"
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}

			var allSum float64 // общая сумма
			msg := "======= Баланс ======="

			USDTmode := true
			if words[1] == "RUB" {
				USDTmode = false
			}

			for key, value := range db[update.Message.Chat.ID] { // цикл перевода валют в $/RUB
				coinPrice, err := whatPriceNow(key, USDTmode) // получение курса
				if err != nil {                               // обработка ошибки
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				}
				dollars := value * coinPrice // валюта * курс
				msg += "\n		" + key + ": " + fmt.Sprint(value) + " (" + fmt.Sprint(dollars) + ")"
				allSum += dollars
			}

			msg += "\nВсего: " + fmt.Sprint(allSum)
			if USDTmode {
				msg += "$"
			} else {
				msg += "RUB"
			}
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))

		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry! I don't know this command D:"))

		}

	}
}

func whatPriceNow(coin string, USDTmode bool) (price float64, err error) { // функция обращения к API для получения курса
	mode := "USDT"
	if !USDTmode {
		mode = "RUB"
	}
	//resp, err := http.Get(fmt.Sprintf("https://www.binance.com/api/v3/ticker/price?symbol=%s%f", coin, mode)) // получение данных с API в долларах
	resp, err := http.Get("https://www.binance.com/api/v3/ticker/price?symbol=" + coin + mode) // получение данных с API в долларах
	if err != nil {
		return
	}

	defer resp.Body.Close() // defer - выполняется только после конца функции, то есть
	// в данном случае мы закрываем обращение к API при выходе из функции
	var jsonBiResp binanceResp // переменная пользовательского типа (описан перед функцией main)
	err = json.NewDecoder(resp.Body).Decode(&jsonBiResp)
	if err != nil { // обработка ошибки
		return
	}
	if jsonBiResp.Code != 0 { // обработка ошибки
		err = errors.New("Некорректная валюта: " + coin)
	}
	price = jsonBiResp.Price // присваивание курса

	return
}

func isExist(coin string) (ex bool) { // проверка действительности валюты
	ex = false
	resp, err := http.Get(fmt.Sprintf("https://www.binance.com/api/v3/ticker/price?symbol=%sUSDT", coin))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonBiResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonBiResp)

	if err == nil && jsonBiResp.Code == 0 {
		ex = true
	}

	return
}

// ===========================================================
