package googlesheets

import (
	"log"
	"strconv"
)

// QuestItem reprezentuje pojedynczy element w questcie
type QuestItem struct {
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes"`
}

// Quest reprezentuje zagregowany element z arkusza Google Sheets
type Quest struct {
	Recipient    string      `json:"recipient"`
	DeliveryDate string      `json:"delivery_date"`
	Location     string      `json:"location"`
	Pavilion     string      `json:"pavilion"`
	Items        []QuestItem `json:"items"`
}

// MapHeaders tłumaczy nagłówki z arkusza na angielskie nazwy pól
func MapHeaders(headers []interface{}) map[int]string {
	headerMap := make(map[int]string)

	for i, header := range headers {
		headerStr, ok := header.(string)
		if !ok {
			continue
		}

		switch headerStr {
		case "Rzeczy":
			headerMap[i] = "item_name"
		case "Ilość":
			headerMap[i] = "quantity"
		case "Pawilon":
			headerMap[i] = "pavilion"
		case "Miejsce":
			headerMap[i] = "location"
		case "Stan":
			headerMap[i] = "status"
		case "Dostawa do":
			headerMap[i] = "delivery_date"
		case "Osoba odpowiedzialna za budżet":
			headerMap[i] = "budget_responsible"
		case "Do kogo ma trafić":
			headerMap[i] = "recipient"
		case "UWAGI":
			headerMap[i] = "notes"
		}
	}

	return headerMap
}

// ParseQuests parsuje dane z arkusza na listę zagregowanych obiektów Quest
func ParseQuests(values [][]interface{}) []Quest {
	log.Printf("Rozpoczynam parsowanie %d wierszy danych", len(values))

	if len(values) < 2 {
		log.Printf("Za mało wierszy danych, potrzebne minimum 2 wiersze (nagłówki + dane)")
		return []Quest{}
	}

	headers := values[0]
	log.Printf("Nagłówki: %v", headers)

	headerMap := MapHeaders(headers)
	log.Printf("Zmapowane nagłówki: %v", headerMap)

	// Map do przechowywania zagregowanych questów
	questMap := make(map[string]*Quest)

	for i := 1; i < len(values); i++ {
		row := values[i]
		log.Printf("Przetwarzanie wiersza %d: %v", i, row)

		var recipient, deliveryDate, location, pavilion, itemName, notes string
		var quantity int

		for j, cell := range row {
			fieldName, exists := headerMap[j]
			if !exists {
				continue
			}

			cellStr, ok := cell.(string)
			if !ok {
				continue
			}

			log.Printf("  Kolumna %d: %s = %s", j, fieldName, cellStr)

			switch fieldName {
			case "recipient":
				recipient = cellStr
			case "delivery_date":
				deliveryDate = cellStr
			case "location":
				location = cellStr
			case "pavilion":
				pavilion = cellStr
			case "item_name":
				itemName = cellStr
			case "quantity":
				q, err := strconv.Atoi(cellStr)
				if err == nil {
					quantity = q
				}
			case "notes":
				notes = cellStr
			}
		}

		// Tworzymy klucz dla mapy na podstawie kombinacji pól
		key := recipient + "|" + deliveryDate + "|" + location + "|" + pavilion

		// Sprawdzamy czy quest już istnieje
		quest, exists := questMap[key]
		if !exists {
			// Tworzymy nowy quest
			quest = &Quest{
				Recipient:    recipient,
				DeliveryDate: deliveryDate,
				Location:     location,
				Pavilion:     pavilion,
				Items:        make([]QuestItem, 0),
			}
			questMap[key] = quest
		}

		// Dodajemy element do questu
		quest.Items = append(quest.Items, QuestItem{
			ItemName: itemName,
			Quantity: quantity,
			Notes:    notes,
		})
	}

	// Konwertujemy mapę na slice
	quests := make([]Quest, 0, len(questMap))
	for _, quest := range questMap {
		quests = append(quests, *quest)
	}

	log.Printf("Zakończono parsowanie, utworzono %d zagregowanych questów", len(quests))
	return quests
}
