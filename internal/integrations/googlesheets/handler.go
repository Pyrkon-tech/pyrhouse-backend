package googlesheets

import (
	"context"
	"fmt"
	"log"
	"os"
	"warehouse/pkg/security"

	"google.golang.org/api/sheets/v4"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleSheetsHandler struct {
	sheetsService *sheets.Service
}

func NewGoogleSheetsHandler() (*GoogleSheetsHandler, error) {
	ctx := context.Background()

	// Sprawdź, czy mamy poświadczenia w zmiennej środowiskowej
	credentialsJSON := os.Getenv("GOOGLE_SHEETS_CREDENTIALS_JSON")
	var credentials *google.Credentials
	var err error

	if credentialsJSON != "" {
		// Użyj poświadczeń z zmiennej środowiskowej
		log.Println("Używam poświadczeń Google z zmiennej środowiskowej")
		credentials, err = google.CredentialsFromJSON(ctx, []byte(credentialsJSON), sheets.SpreadsheetsScope)
	} else {
		// Użyj pliku lokalnego (tylko dla środowiska deweloperskiego)
		log.Println("Używam poświadczeń Google z pliku lokalnego")
		credentialsFile := "configs/google-credentials.json"
		b, err := os.ReadFile(credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("nie można odczytać pliku z danymi uwierzytelniającymi: %v", err)
		}
		credentials, err = google.CredentialsFromJSON(ctx, b, sheets.SpreadsheetsScope)
	}

	if err != nil {
		return nil, fmt.Errorf("nie można załadować poświadczeń Google: %v", err)
	}

	client := oauth2.NewClient(ctx, credentials.TokenSource)
	sheetsService, err := sheets.New(client)
	if err != nil {
		return nil, fmt.Errorf("nie można utworzyć klienta Google Sheets: %v", err)
	}

	return &GoogleSheetsHandler{
		sheetsService: sheetsService,
	}, nil
}

func (h *GoogleSheetsHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/sheets/quests", security.Authorize("moderator"), h.getQuests)
}

func (h *GoogleSheetsHandler) getQuests(c *gin.Context) {
	spreadsheetID := "1mWc7g905RxTmBfEnzvtwNUjQXkeDsqzD8J79WsOEex4"
	readRange := "A1:I999"

	if spreadsheetID == "" || readRange == "" {
		c.JSON(400, gin.H{
			"error": "Wymagane parametry spreadsheet_id i range",
		})
		return
	}

	values, err := h.ReadSpreadsheet(spreadsheetID, readRange)
	if err != nil {
		log.Printf("Błąd podczas pobierania danych: %v", err)
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	if values == nil {
		log.Printf("Nie znaleziono danych w arkuszu")
		c.JSON(200, gin.H{
			"quests": []Quest{},
		})
		return
	}

	quests := ParseQuests(values)
	log.Printf("Przetworzono %d questów", len(quests))

	c.JSON(200, quests)
}

func (h *GoogleSheetsHandler) ReadSpreadsheet(spreadsheetID string, readRange string) ([][]interface{}, error) {
	resp, err := h.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("nie można odczytać arkusza: %v", err)
	}

	if len(resp.Values) == 0 {
		log.Printf("Nie znaleziono danych w zakresie %s", readRange)
		return nil, nil
	}

	return resp.Values, nil
}
