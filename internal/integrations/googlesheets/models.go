package googlesheets

import (
	"log"
	"strconv"
)

// QuestItem represents a single item in a quest
type QuestItem struct {
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes"`
	Status   string `json:"status"`
}

// Quest represents an aggregated element from a Google Sheets spreadsheet
type Quest struct {
	Recipient    string      `json:"recipient"`
	DeliveryDate string      `json:"delivery_date"`
	Location     string      `json:"location"`
	Pavilion     string      `json:"pavilion"`
	Status       string      `json:"status"`
	Items        []QuestItem `json:"items"`
}

// MapHeaders maps spreadsheet headers to English field names
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

// ParseQuests parses spreadsheet data into a list of aggregated Quest objects
func ParseQuests(values [][]interface{}) []Quest {
	log.Printf("[spreadsheet-parser] Starting to parse %d data rows", len(values))

	if len(values) < 2 {
		log.Printf("[spreadsheet-parser] Not enough data rows, minimum 2 rows required (headers + data)")
		return []Quest{}
	}

	headers := values[0]
	log.Printf("[spreadsheet-parser] Headers: %v", headers)

	headerMap := MapHeaders(headers)
	log.Printf("[spreadsheet-parser] Mapped headers: %v", headerMap)

	// Map for storing aggregated quests
	questMap := make(map[string]*Quest)

	for i := 1; i < len(values); i++ {
		row := values[i]
		log.Printf("[spreadsheet-parser] Processing row %d: %v", i, row)

		var recipient, deliveryDate, location, pavilion, itemName, notes, status string
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

			log.Printf("[spreadsheet-parser] Kolumna %d: %s = %s", j, fieldName, cellStr)

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
			case "status":
				status = cellStr
			}
		}

		// Create a map key based on the combination of fields
		key := recipient + "|" + deliveryDate + "|" + location + "|" + pavilion

		// Check if the quest already exists
		quest, exists := questMap[key]
		if !exists {
			// Create a new quest
			quest = &Quest{
				Recipient:    recipient,
				DeliveryDate: deliveryDate,
				Location:     location,
				Pavilion:     pavilion,
				Status:       status,
				Items:        make([]QuestItem, 0),
			}
			questMap[key] = quest
		}

		// Add item to the quest
		quest.Items = append(quest.Items, QuestItem{
			ItemName: itemName,
			Quantity: quantity,
			Notes:    notes,
			Status:   status,
		})
	}

	quests := make([]Quest, 0, len(questMap))
	for _, quest := range questMap {
		quests = append(quests, *quest)
	}

	log.Printf("[spreadsheet-parser] Parsing completed, created %d aggregated quests", len(quests))
	return quests
}
