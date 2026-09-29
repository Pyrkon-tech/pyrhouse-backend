# Fixtures dla systemu magazynowego

`fixtures.sql` to dane testowe zbudowane ze zrzutu bazy staging i zanonimizowane — do środowiska deweloperskiego,
podglądu i testów e2e.

## Jak użyć

Na zmigrowanej bazie (plik zaczyna się od `TRUNCATE` wszystkich tabel poza `schema_migrations`, więc zastępuje ich
zawartość):

```sh
docker exec -i go-test-db-postgres-1 psql -U postgres -d <baza> < postgres/data.sql/fixtures.sql
```

Hasło każdego konta: `pyrhouse`. Pierwszy aktywny admin nazywa się `admin`, pozostali `user<id>`.

## Jak przebudować

Z nowszego zrzutu (plik `.sql` z `pg_dump`, trzymany poza repo — jest w `.gitignore`):

```sh
sh postgres/build-fixtures.sh staging-pyrhouse-<data>-restore.sql
```

Skrypt odtwarza zrzut w tymczasowej bazie w Postgresie z docker-compose, anonimizuje go (`postgres/anonymize.sql`),
migruje do bieżącego schematu, zrzuca same dane do `fixtures.sql` i usuwa tymczasową bazę. Niczego ze zrzutu nie
wypisuje.

## Co jest anonimizowane

- użytkownicy: login, imię i nazwisko, hasło, Discord (id, nazwa), Google (id, e-mail), avatar;
- odbiorcy transferów i zapotrzebowań (także w kluczu `quest_key`), właściciele budżetów, geolokalizacja dostaw
  (przesunięta na teren MTP);
- wolontariusze: nick, miasto, notatki, potwierdzenie Discord (to pole trzyma nazwę użytkownika Discorda);
- wolny tekst: zgłoszenia i komentarze Service Desk, notatki wydań i pozycji zapotrzebowań;
- identyfikatory arkuszy Google w ustawieniach i logach synchronizacji; `audit_logs.data` wyczyszczone, logi przycięte.

Zamiana jest deterministyczna: ta sama oryginalna wartość dostaje zawsze to samo zastępcze imię i nazwisko, więc
jeden odbiorca pozostaje jedną osobą w wielu transferach.

Nie są zmieniane: kategorie, lokalizacje, sprzęt (numery seryjne, kody PYR), stany magazynowe, grafik (sloty,
przypisania), cennik i dostawcy.

**Przy nowej kolumnie z danymi osobowymi dopisz ją do `anonymize.sql`.**
