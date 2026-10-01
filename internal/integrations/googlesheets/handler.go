package googlesheets

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/sheets/v4"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleSheetsHandler struct {
	sheetsService *sheets.Service
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
		sheetsService: sheetsService,
	}, nil
}

func (h *GoogleSheetsHandler) ClearSheet(spreadsheetID, sheetName string) error {
	_, err := h.sheetsService.Spreadsheets.Values.Clear(
		spreadsheetID, sheetName, &sheets.ClearValuesRequest{},
	).Do()
	if err != nil {
		return fmt.Errorf("unable to clear sheet: %v", err)
	}
	return nil
}

func (h *GoogleSheetsHandler) WriteSpreadsheet(spreadsheetID, writeRange string, values [][]interface{}) error {
	vr := &sheets.ValueRange{
		Values: values,
	}
	_, err := h.sheetsService.Spreadsheets.Values.Update(
		spreadsheetID, writeRange, vr,
	).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("unable to write spreadsheet: %v", err)
	}
	return nil
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
