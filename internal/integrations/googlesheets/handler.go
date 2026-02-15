package googlesheets

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"warehouse/internal/security"

	"google.golang.org/api/sheets/v4"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleSheetsHandler struct {
	sheetsService       *sheets.Service
	dutyScheduleService *DutyScheduleService
}

func NewGoogleSheetsHandler() (*GoogleSheetsHandler, error) {
	ctx := context.Background()

	// Check if we have credentials in the environment variable
	credentialsJSON := os.Getenv("GOOGLE_SHEETS_CREDENTIALS_JSON")
	var credentials *google.Credentials
	var err error

	if credentialsJSON != "" {
		// Use credentials from environment variable
		log.Println("Using Google credentials from environment variable")
		credentials, err = google.CredentialsFromJSON(ctx, []byte(credentialsJSON), sheets.SpreadsheetsScope)
	} else {
		// Use local file (development environment only)
		log.Println("Using Google credentials from local file")
		credentialsFile := "configs/google-credentials.json"
		b, err := os.ReadFile(credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read credentials file: %v", err)
		}
		credentials, err = google.CredentialsFromJSON(ctx, b, sheets.SpreadsheetsScope)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to load Google credentials: %v", err)
	}

	client := oauth2.NewClient(ctx, credentials.TokenSource)
	sheetsService, err := sheets.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to create Google Sheets client: %v", err)
	}

	return &GoogleSheetsHandler{
		sheetsService:       sheetsService,
		dutyScheduleService: NewDutyScheduleService(sheetsService),
	}, nil
}

func (h *GoogleSheetsHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/sheets/quests", security.Authorize("user"), h.getQuests)
	router.GET("/sheets/duty-schedule", h.getDutySchedule)
}

func (h *GoogleSheetsHandler) getQuests(c *gin.Context) {
	spreadsheetID := "1mWc7g905RxTmBfEnzvtwNUjQXkeDsqzD8J79WsOEex4"
	readRange := "A1:I999"

	if spreadsheetID == "" || readRange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Required parameters: spreadsheet_id and range",
		})
		return
	}

	filterStatus := c.Query("status")
	values, err := h.ReadSpreadsheet(spreadsheetID, readRange)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if values == nil {
		log.Printf("No data found in the spreadsheet")
		c.JSON(http.StatusOK, []Quest{})
		return
	}

	quests := ParseQuests(values)
	filteredQuests := filterQuestsByStatus(quests, filterStatus)
	log.Printf("Przetworzono %d questów, po filtrowaniu: %d", len(quests), len(filteredQuests))

	c.JSON(http.StatusOK, filteredQuests)
}

func filterQuestsByStatus(quests []Quest, status string) []Quest {
	if len(quests) == 0 {
		return []Quest{}
	}

	filtered := make([]Quest, 0, len(quests))
	for _, quest := range quests {
		if status == "delivered" && quest.Status == "Wysłane" {
			filtered = append(filtered, quest)
		} else if status == "" && (quest.Status == "Zamówione" || quest.Status == "Zatwierdzone") {
			filtered = append(filtered, quest)
		}
	}
	return filtered
}

func (h *GoogleSheetsHandler) ReadSpreadsheet(spreadsheetID string, readRange string) ([][]interface{}, error) {
	resp, err := h.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to read spreadsheet: %v", err)
	}

	if len(resp.Values) == 0 {
		log.Printf("No data found in range %s", readRange)
		return nil, nil
	}

	return resp.Values, nil
}

func (h *GoogleSheetsHandler) getDutySchedule(c *gin.Context) {
	spreadsheetID := "11kikoxFRrhDiHJJNSvAky6kk8eblFO4jL6n3Hj3FPwo"
	readRange := "G1:CH12" // Adjust range as needed

	values, err := h.ReadSpreadsheet(spreadsheetID, readRange)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if values == nil {
		log.Printf("No data found in the spreadsheet")
		c.JSON(http.StatusOK, [][]interface{}{})
		return
	}

	c.JSON(http.StatusOK, values)
}
