# Plan: Integracja grafiku dyżurów z Google Spreadsheet

## Problem

Obecny eksport grafiku to CSV (`GET /schedules/:id/export`). CSV trzeba pobrać, otworzyć w Excelu, ręcznie poprawić formatowanie. Nie ma możliwości wizualnej edycji online ani współdzielenia z zespołem.

## Cel

Zamiast (lub obok) CSV — pushować grafik bezpośrednio do Google Spreadsheet, gdzie:
- Dane ładują się do konkretnych komórek
- Moderator może wizualnie poprawić ręcznie w przeglądarce
- Każde wywołanie "Generuj" / "Eksportuj" nadpisuje arkusz aktualnym stanem

## Docelowy arkusz

- **Spreadsheet ID:** `1vq1l8gYn9rMDcdm5XvSZUNW48RRok4pESZA1Sk0Q60U`
- **Nazwa karty:** `grafik`
- Konfigurowane przez `app_settings` (jak equipment requests), nie hardcoded

## Analiza obecnego stanu

### Co już mamy

| Element | Status | Uwagi |
|---------|--------|-------|
| Google Sheets client (`sheetsService`) | **gotowy** | `internal/integrations/googlesheets/handler.go` |
| Credentials (env + plik) | **gotowe** | `GOOGLE_SHEETS_CREDENTIALS_JSON` lub `configs/google-credentials.json` |
| OAuth scope | **gotowy do zapisu** | Używamy `sheets.SpreadsheetsScope` = `googleapis.com/auth/spreadsheets` (read+write!) |
| `ReadSpreadsheet()` | **gotowy** | Czyta dowolny range |
| **`WriteSpreadsheet()`** | **brak** | Trzeba dodać |
| **`BatchUpdate()` (formatowanie)** | **brak** | Trzeba dodać |
| Settings: dynamiczny sheet ID | **wzorzec gotowy** | `equipment_request.sheet_id` — ten sam pattern |
| Export logic (dane grafiku) | **gotowy** | `internal/scheduling/export.go` — generuje dane, trzeba zmienić target |

### Kluczowe odkrycie

Scope `sheets.SpreadsheetsScope` to pełny read/write — **nie trzeba zmieniać credentials ani scope**. Service account musi mieć dostęp do docelowego arkusza (udostępnić arkusz dla email service account).

## Proponowana architektura

### Układ komórek w arkuszu "grafik"

```
     A              B              C              D              E         ...
1    Godzina        Stanowisko 1   Stanowisko 2   Stanowisko 3   Stanowisko 4
2    --- Montaż Wtorek ---
3    cały dzień     Kozak          Pixel          Rocky
4    --- Montaż Środa ---
5    cały dzień     Kozak          Max
6    --- Montaż Czwartek ---
7    cały dzień     Pixel          Rocky
8    --- Piątek ---
9    10:00-11:00    Kozak          Pixel          Rocky          Max
10   11:00-12:00    Kozak          Pixel          Rocky          Max
...
70   --- Sobota ---
71   00:00-01:00    Nocny1         Nocny2
...
140  --- Niedziela ---
...
200  19:00-20:00    Kozak          Rocky
201  --- Demontaż Poniedziałek ---
202  cały dzień     Kozak          Pixel          Rocky
```

### Formatowanie (opcjonalne, faza 2)

- **Nagłówki sekcji** (--- Montaż ---): bold, szare tło, merge cells A-E
- **Nagłówek tabeli** (Stanowisko 1..N): bold, zamrożony wiersz
- **Komórki z pseudonimami**: zwykły tekst
- **Nocne godziny** (00:00-06:00): ciemniejsze tło
- **Kolumny**: auto-resize do zawartości

## Plan implementacji

### Faza 1: Write do Google Sheets (MVP)

#### 1.1 Dodać `WriteSpreadsheet()` do GoogleSheetsHandler

```go
// handler.go — nowa metoda
func (h *GoogleSheetsHandler) WriteSpreadsheet(spreadsheetID, writeRange string, values [][]interface{}) error {
    vr := &sheets.ValueRange{
        Values: values,
    }
    _, err := h.sheetsService.Spreadsheets.Values.Update(
        spreadsheetID, writeRange, vr,
    ).ValueInputOption("RAW").Do()
    return err
}
```

Opcjonalnie też:

```go
// Czyści cały arkusz przed zapisem
func (h *GoogleSheetsHandler) ClearSheet(spreadsheetID, sheetName string) error {
    _, err := h.sheetsService.Spreadsheets.Values.Clear(
        spreadsheetID, sheetName, &sheets.ClearValuesRequest{},
    ).Do()
    return err
}
```

#### 1.2 Dodać settings w `app_settings`

Nowa migracja (seed data) lub ręcznie przez API:

```sql
INSERT INTO app_settings (key, value, description) VALUES
('scheduling.sheet_id', '1vq1l8gYn9rMDcdm5XvSZUNW48RRok4pESZA1Sk0Q60U', 'Google Spreadsheet ID for schedule export'),
('scheduling.sheet_name', 'grafik', 'Sheet/tab name for schedule export');
```

Moderator może zmienić przez `PUT /settings/scheduling.sheet_id`.

#### 1.3 Zmienić `export.go` → `sheets_export.go`

Zamiast zwracać CSV string, przygotować `[][]interface{}` (format Google Sheets API) i wywołać write:

```go
// internal/scheduling/sheets_export.go

func ExportToSheet(
    sheetsHandler *googlesheets.GoogleSheetsHandler,
    sheetID, sheetName string,
    schedule *Schedule,
    slots []Slot,
    volunteers []Volunteer,
    assignments []Assignment,
) error {
    // 1. Przygotuj dane (ta sama logika co CSV, ale jako [][]interface{})
    rows := buildSheetRows(schedule, slots, volunteers, assignments)

    // 2. Wyczyść arkusz
    sheetsHandler.ClearSheet(sheetID, sheetName)

    // 3. Zapisz dane
    writeRange := fmt.Sprintf("%s!A1", sheetName)
    return sheetsHandler.WriteSpreadsheet(sheetID, writeRange, rows)
}
```

#### 1.4 Nowy endpoint lub zmiana istniejącego

Dwie opcje:

**Opcja A: Nowy endpoint** (rekomendowana)
```
POST /schedules/:id/export/sheets    (moderator)
```
Response: `{ "url": "https://docs.google.com/spreadsheets/d/XXX/edit", "rows_written": 205 }`

**Opcja B: Query param na istniejącym**
```
GET /schedules/:id/export?format=sheets
GET /schedules/:id/export?format=csv     (domyślnie)
```

#### 1.5 Wiring w DI

- `SchedulingService` potrzebuje dostępu do `GoogleSheetsHandler` i `SettingsRepository`
- Dodać do konstruktora Service lub Handler

```go
type Service struct {
    repo          *Repository
    sheetsHandler *googlesheets.GoogleSheetsHandler  // może być nil
    settingsRepo  *settings.Repository
}
```

#### 1.6 Udostępnić arkusz dla service account

Service account email (z `google-credentials.json`, pole `client_email`) musi mieć dostęp "Edytor" do docelowego arkusza. Jednorazowa czynność ręczna — w Google Sheets kliknij "Udostępnij" i dodaj email service account.

### Faza 2: Formatowanie (opcjonalna)

Użyć `Spreadsheets.BatchUpdate()` z requestami:

```go
func (h *GoogleSheetsHandler) FormatSheet(spreadsheetID string, requests []*sheets.Request) error {
    batchReq := &sheets.BatchUpdateSpreadsheetRequest{
        Requests: requests,
    }
    _, err := h.sheetsService.Spreadsheets.BatchUpdate(spreadsheetID, batchReq).Do()
    return err
}
```

Requesty do formatowania:
- `RepeatCellRequest` — bold/kolory na nagłówkach sekcji
- `MergeCellsRequest` — merge komórek dla nagłówków "--- Piątek ---"
- `UpdateSheetPropertiesRequest` — zamrożenie pierwszego wiersza
- `AutoResizeDimensionsRequest` — auto-width kolumn
- `UpdateDimensionPropertiesRequest` — szerokość kolumny A

### Faza 3: Dwukierunkowy sync (przyszłość, niska priorytet)

Czytanie zmian z arkusza z powrotem do systemu:
1. Moderator edytuje pseudonim w komórce (zamiana ręczna)
2. Endpoint `POST /schedules/:id/import/sheets` czyta arkusz
3. Parsuje zmiany i aktualizuje assignments w DB

Wymaga: rozpoznawania pseudonimów, mapowania komórek na sloty. Skomplikowane — lepiej zostawić ręczne swapy przez API.

## Pliki do zmiany/utworzenia

| Plik | Zmiana |
|------|--------|
| `internal/integrations/googlesheets/handler.go` | Dodać `WriteSpreadsheet()`, `ClearSheet()` |
| `internal/scheduling/sheets_export.go` | **NOWY** — logika eksportu do Google Sheets |
| `internal/scheduling/service.go` | Dodać `ExportToSheets()`, nowe zależności w konstruktorze |
| `internal/scheduling/handler.go` | Dodać endpoint `POST /schedules/:id/export/sheets` |
| `internal/di/container.go` | Przekazać `GoogleSheetsHandler` + `SettingsRepo` do scheduling |
| `internal/scheduling/export.go` | Refactor: wyciągnąć wspólną logikę budowania wierszy |

## Estymacja złożoności

| Faza | Zakres |
|------|--------|
| Faza 1 (Write MVP) | ~6 plików, nowa metoda write + endpoint + wiring |
| Faza 2 (Formatowanie) | +1 metoda `FormatSheet()`, ~50 linii requestów formatowania |
| Faza 3 (Dwukierunkowy sync) | Duży scope — osobny plan |

## Ryzyka i uwagi

1. **Service account musi mieć dostęp do arkusza** — jednorazowe ręczne udostępnienie
2. **Rate limits Google Sheets API** — 300 req/min na projekt, nie problem przy manualnych eksportach
3. **Scope już jest read/write** — nie trzeba zmieniać credentials
4. **Istniejący CSV export zostaje** — sheets export to dodatkowa opcja, nie zastępstwo
5. **Jeśli GoogleSheetsHandler = nil** (brak credentials) → endpoint zwraca 503 z komunikatem
6. **Nadpisywanie danych** — każdy eksport czyści arkusz i pisze od nowa (najprostsze i najbezpieczniejsze)
