package googlesheets

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/sheets/v4"
)

type DutyScheduleService struct {
	sheetsService *sheets.Service
}

func NewDutyScheduleService(sheetsService *sheets.Service) *DutyScheduleService {
	return &DutyScheduleService{
		sheetsService: sheetsService,
	}
}

type DutySchedule struct {
	WarehouseName string            `json:"warehouse_name"`
	Installation  string            `json:"installation"`
	Tuesday       string            `json:"tuesday"`
	Wednesday     string            `json:"wednesday"`
	Thursday      string            `json:"thursday"`
	Friday        map[string]string `json:"friday"`
	Saturday      map[string]string `json:"saturday"`
	Sunday        map[string]string `json:"sunday"`
}

type DutyTimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Person    string    `json:"person"`
}

type DutyScheduleResponse struct {
	WarehouseName string         `json:"warehouse_name"`
	Installation  string         `json:"installation"`
	Schedule      []DutyTimeSlot `json:"schedule"`
}

func (s *DutyScheduleService) GetDutySchedule() ([]DutySchedule, error) {
	spreadsheetID := "11kikoxFRrhDiHJJNSvAky6kk8eblFO4jL6n3Hj3FPwo"
	readRange := "G1:CH12"

	values, err := s.readSpreadsheet(spreadsheetID, readRange)
	if err != nil {
		return nil, fmt.Errorf("nie można odczytać arkusza: %v", err)
	}

	if values == nil {
		log.Printf("Nie znaleziono danych w arkuszu")
		return []DutySchedule{}, nil
	}

	return s.parseDutySchedule(values), nil
}

func (s *DutyScheduleService) GetDutyScheduleForPerson(personName string) ([]DutyScheduleResponse, error) {
	schedules, err := s.GetDutySchedule()
	if err != nil {
		return nil, err
	}

	var result []DutyScheduleResponse
	for _, schedule := range schedules {
		response := DutyScheduleResponse{
			WarehouseName: schedule.WarehouseName,
			Installation:  schedule.Installation,
			Schedule:      make([]DutyTimeSlot, 0),
		}

		// Sprawdź dyżury na wtorek
		if schedule.Tuesday == personName {
			response.Schedule = append(response.Schedule, DutyTimeSlot{
				StartTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 11, 0, 0, 0, time.Local),
				EndTime:   time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 23, 0, 0, 0, time.Local),
				Person:    personName,
			})
		}

		// Sprawdź dyżury na środę
		if schedule.Wednesday == personName {
			response.Schedule = append(response.Schedule, DutyTimeSlot{
				StartTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 11, 0, 0, 0, time.Local),
				EndTime:   time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 23, 0, 0, 0, time.Local),
				Person:    personName,
			})
		}

		// Sprawdź dyżury na czwartek
		if schedule.Thursday == personName {
			response.Schedule = append(response.Schedule, DutyTimeSlot{
				StartTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 11, 0, 0, 0, time.Local),
				EndTime:   time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 23, 0, 0, 0, time.Local),
				Person:    personName,
			})
		}

		// Sprawdź dyżury na piątek
		for timeSlot, person := range schedule.Friday {
			if person == personName {
				startTime, endTime := parseTimeSlot(timeSlot)
				response.Schedule = append(response.Schedule, DutyTimeSlot{
					StartTime: startTime,
					EndTime:   endTime,
					Person:    personName,
				})
			}
		}

		// Sprawdź dyżury na sobotę
		for timeSlot, person := range schedule.Saturday {
			if person == personName {
				startTime, endTime := parseTimeSlot(timeSlot)
				response.Schedule = append(response.Schedule, DutyTimeSlot{
					StartTime: startTime,
					EndTime:   endTime,
					Person:    personName,
				})
			}
		}

		// Sprawdź dyżury na niedzielę
		for timeSlot, person := range schedule.Sunday {
			if person == personName {
				startTime, endTime := parseTimeSlot(timeSlot)
				response.Schedule = append(response.Schedule, DutyTimeSlot{
					StartTime: startTime,
					EndTime:   endTime,
					Person:    personName,
				})
			}
		}

		if len(response.Schedule) > 0 {
			result = append(result, response)
		}
	}

	return result, nil
}

func (s *DutyScheduleService) readSpreadsheet(spreadsheetID string, readRange string) ([][]interface{}, error) {
	resp, err := s.sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("nie można odczytać arkusza: %v", err)
	}

	if len(resp.Values) == 0 {
		log.Printf("Nie znaleziono danych w zakresie %s", readRange)
		return nil, nil
	}

	return resp.Values, nil
}

func (s *DutyScheduleService) parseDutySchedule(values [][]interface{}) []DutySchedule {
	if len(values) < 2 {
		return []DutySchedule{}
	}

	schedules := make([]DutySchedule, 0)

	// Pomijamy pierwszy wiersz (nagłówki)
	for i := 1; i < len(values); i++ {
		row := values[i]
		if len(row) < 2 {
			continue
		}

		schedule := DutySchedule{
			WarehouseName: toString(row[0]),
			Installation:  toString(row[1]),
			Tuesday:       toString(row[2]),
			Wednesday:     toString(row[3]),
			Thursday:      toString(row[4]),
			Friday:        make(map[string]string),
			Saturday:      make(map[string]string),
			Sunday:        make(map[string]string),
		}

		// Parsowanie godzin dla piątku (indeksy 6-29)
		for j := 6; j < 30; j++ {
			if j < len(row) {
				timeSlot := values[0][j].(string)
				schedule.Friday[timeSlot] = toString(row[j])
			}
		}

		// Parsowanie godzin dla soboty (indeksy 30-53)
		for j := 30; j < 54; j++ {
			if j < len(row) {
				timeSlot := values[0][j].(string)
				schedule.Saturday[timeSlot] = toString(row[j])
			}
		}

		// Parsowanie godzin dla niedzieli (indeksy 54-77)
		for j := 54; j < 78; j++ {
			if j < len(row) {
				timeSlot := values[0][j].(string)
				schedule.Sunday[timeSlot] = toString(row[j])
			}
		}

		schedules = append(schedules, schedule)
	}

	return schedules
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func parseTimeSlot(timeSlot string) (time.Time, time.Time) {
	// Przykładowy format: "00:00 - 01:00"
	layout := "15:04"
	parts := splitTimeSlot(timeSlot)

	startTime, _ := time.Parse(layout, parts[0])
	endTime, _ := time.Parse(layout, parts[1])

	now := time.Now()
	startTime = time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, time.Local)
	endTime = time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, time.Local)

	return startTime, endTime
}

func splitTimeSlot(timeSlot string) []string {
	// Usuń spacje i podziel na części
	parts := make([]string, 2)
	timeSlot = removeSpaces(timeSlot)
	parts[0] = timeSlot[:5]
	parts[1] = timeSlot[7:]
	return parts
}

func removeSpaces(s string) string {
	var result string
	for _, char := range s {
		if char != ' ' {
			result += string(char)
		}
	}
	return result
}
