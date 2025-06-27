# Fixtures dla systemu magazynowego

## Opis

Plik `fixtures.sql` zawiera dane testowe dla systemu magazynowego Pyrkon. Dane zostały pobrane z produkcji i zanonimizowane w celu bezpiecznego użycia w środowisku deweloperskim.

## Zawartość

Plik zawiera następujące dane:

### Kategorie przedmiotów (item_category)
- 37 różnych kategorii przedmiotów
- Podział na typy: `asset` (przedmioty seryjne) i `stock` (przedmioty nieseryjne)
- Przykłady: laptopy, drukarki, telewizory, kable, przedłużacze

### Lokalizacje (locations)
- 41 różnych lokalizacji na terenie festiwalu
- Zawiera informacje o pawilonach i szczegółach lokalizacji
- Przykłady: Magazyn Techniczny, HQ, RedRoom, Biuro Akredytacji

### Użytkownicy (users)
- 31 użytkowników z różnymi rolami
- Role: admin, moderator, user
- Hasła są zanonimizowane (hash bcrypt)

### Przedmioty seryjne (items)
- 120 przedmiotów seryjnych (assets)
- Różne statusy: located, available
- Przykłady: laptopy, drukarki, telewizory, tablety, telefony

### Przedmioty nieseryjne (non_serialized_items)
- 27 pozycji magazynowych (stock items)
- Różne ilości i lokalizacje
- Przykłady: przedłużacze, kable, skanery kodów

## Użycie

### Import do bazy danych

```bash
# Po uruchomieniu migracji
psql -d warehouse -f postgres/data.sql/fixtures.sql
```

### W Docker Compose

```yaml
# W docker-compose.yml
services:
  postgres:
    volumes:
      - ./postgres/data.sql/fixtures.sql:/docker-entrypoint-initdb.d/02-fixtures.sql
```

### W kodzie Go

```go
// W main.go lub setup
db.Exec("\\i postgres/data.sql/fixtures.sql")
```

## Struktura danych

### Kategorie przedmiotów
- `id` - unikalny identyfikator
- `item_category` - nazwa kategorii (unique)
- `label` - etykieta wyświetlana
- `pyr_id` - kod PYR (4 znaki)
- `category_type` - typ: 'asset' lub 'stock'

### Lokalizacje
- `id` - unikalny identyfikator
- `name` - nazwa lokalizacji
- `details` - szczegóły (opcjonalne)
- `pavilion` - numer pawilonu (opcjonalne)

### Użytkownicy
- `id` - unikalny identyfikator
- `username` - nazwa użytkownika (unique)
- `fullname` - pełne imię i nazwisko
- `password_hash` - hash hasła (bcrypt)
- `role` - rola: admin, moderator, user
- `points` - punkty użytkownika
- `active` - czy konto aktywne

### Przedmioty seryjne
- `id` - unikalny identyfikator
- `item_serial` - numer seryjny (opcjonalny)
- `status` - status: located, available, in_transfer, delivered
- `location_id` - ID lokalizacji
- `item_category_id` - ID kategorii
- `pyr_code` - kod PYR (unique)
- `origin` - źródło: probis, netland, druga-era, oki-event, other-mortis

### Przedmioty nieseryjne
- `id` - unikalny identyfikator
- `item_category_id` - ID kategorii
- `location_id` - ID lokalizacji
- `quantity` - ilość
- `origin` - źródło
- `status` - status (opcjonalny)

## Uwagi

1. **Sekwencje** - Plik automatycznie resetuje sekwencje po wstawieniu danych
2. **Klucze obce** - Wszystkie klucze obce są poprawnie skonfigurowane
3. **Unikalność** - Kody PYR są unikalne dla przedmiotów seryjnych
4. **Anonimizacja** - Dane zostały zanonimizowane, ale zachowują strukturę produkcyjną

## Aktualizacja

Aby zaktualizować fixture z nowymi danymi z produkcji:

1. Eksportuj dane z produkcji
2. Zanonimizuj wrażliwe dane
3. Zaktualizuj plik `fixtures.sql`
4. Przetestuj import w środowisku deweloperskim

## Bezpieczeństwo

- Hasła są zahashowane za pomocą bcrypt
- Dane osobowe zostały zanonimizowane
- Nie zawiera wrażliwych informacji biznesowych
- Może być bezpiecznie używany w środowisku deweloperskim 