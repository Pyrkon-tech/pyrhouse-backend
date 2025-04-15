# Konfiguracja poświadczeń Google

## Środowisko deweloperskie

1. Pobierz plik poświadczeń z konsoli Google Cloud:
   - Przejdź do [Google Cloud Console](https://console.cloud.google.com/)
   - Wybierz swój projekt
   - Przejdź do "APIs & Services" > "Credentials"
   - Utwórz nowe poświadczenia lub użyj istniejących
   - Pobierz plik JSON z poświadczeniami

2. Umieść plik w katalogu `configs/` jako `google-credentials.json`

## Środowisko produkcyjne (DigitalOcean App Platform)

1. W panelu DigitalOcean App Platform:
   - Przejdź do ustawień Twojej aplikacji
   - Znajdź sekcję "Environment Variables"
   - Dodaj nową zmienną o nazwie `GOOGLE_CREDENTIALS_JSON`
   - Wklej całą zawartość pliku `google-credentials.json` jako wartość tej zmiennej

## Bezpieczeństwo

- Nigdy nie wysyłaj pliku `google-credentials.json` do repozytorium
- Plik jest już dodany do `.gitignore`
- Używaj różnych poświadczeń dla środowisk deweloperskiego i produkcyjnego
- Regularnie rotuj poświadczenia w środowisku produkcyjnym

## Rozwiązywanie problemów

Jeśli aplikacja nie może połączyć się z Google Sheets API:
1. Sprawdź, czy plik poświadczeń jest poprawny
2. Upewnij się, że API Google Sheets jest włączone w projekcie Google Cloud
3. Sprawdź, czy poświadczenia mają odpowiednie uprawnienia
4. Zweryfikuj, czy zmienna środowiskowa `GOOGLE_CREDENTIALS_JSON` jest poprawnie ustawiona w środowisku produkcyjnym

# Instrukcja konfiguracji Google Sheets API

Aby poprawnie skonfigurować integrację z Google Sheets API, musisz wykonać następujące kroki:

## 1. Utwórz projekt w Google Cloud Console

1. Przejdź do [Google Cloud Console](https://console.cloud.google.com/)
2. Utwórz nowy projekt lub wybierz istniejący
3. Zapisz ID projektu

## 2. Włącz Google Sheets API

1. W menu bocznym wybierz "APIs & Services" > "Library"
2. Wyszukaj "Google Sheets API"
3. Kliknij "Enable" (Włącz)

## 3. Utwórz konto usługi

1. W menu bocznym wybierz "APIs & Services" > "Credentials"
2. Kliknij "Create Credentials" > "Service Account"
3. Wypełnij formularz:
   - Nazwa konta usługi (np. "pyrhouse")
   - ID konta usługi (zostanie wygenerowane automatycznie)
   - Opis (opcjonalnie)
4. Kliknij "Create and Continue"
5. W sekcji "Grant this service account access to project" możesz pominąć ten krok, klikając "Continue"
6. Kliknij "Done"

## 4. Utwórz klucz dla konta usługi

1. Na liście kont usługi znajdź utworzone konto i kliknij na nie
2. Przejdź do zakładki "Keys"
3. Kliknij "Add Key" > "Create new key"
4. Wybierz format "JSON"
5. Kliknij "Create"
6. Plik z kluczem zostanie automatycznie pobrany na Twój komputer

## 5. Skonfiguruj dostęp do arkusza

1. Otwórz arkusz Google Sheets, do którego chcesz uzyskać dostęp
2. Kliknij przycisk "Share" (Udostępnij) w prawym górnym rogu
3. W polu "Add people and groups" wklej adres e-mail konta usługi (znajdziesz go w pliku JSON w polu `client_email`)
4. Nadaj uprawnienia "Editor" (Edytor) lub "Viewer" (Przeglądający), w zależności od potrzeb
5. Kliknij "Share" (Udostępnij)

## 6. Skonfiguruj aplikację

1. Zmień nazwę pobranego pliku JSON na `google-credentials.json`
2. Umieść plik w katalogu `configs/` w projekcie
3. Upewnij się, że plik zawiera prawidłowe dane (nie przykładowe)

## 7. Uzyskaj ID arkusza

ID arkusza można znaleźć w URL arkusza:
```
https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit
```
Gdzie `SPREADSHEET_ID` to długi ciąg znaków, który należy użyć w zapytaniach API.

## Przykład użycia API

```
GET /api/sheets/read?spreadsheet_id=SPREADSHEET_ID&range=A1:Z10
```

## Rozwiązywanie problemów

Jeśli otrzymujesz błąd "private key should be a PEM or plain PKCS1 or PKCS8", oznacza to, że używasz przykładowego klucza prywatnego zamiast prawdziwego. Upewnij się, że zastąpiłeś przykładowe dane w pliku `google-credentials.json` prawdziwymi danymi z pobranego pliku JSON. 