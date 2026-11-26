# Wydarzenia Giełdowe

Agregator wydarzeń rynkowych na Androida i Windows: indeksy, waluty i surowce, nie tylko DAX. Ten sam interfejs w obu wersjach (`ui/index.html`).

Zbiera wszystko w jeden kalendarz: dane makro, decyzje banków centralnych, wygasanie kontraktów i opcji, rollover, święta giełdowe, skrócone sesje, zmianę czasu USA/Europa i wyniki spółek.

Rynki (każdy włączany osobno w Filtrach): Niemcy / strefa euro (DAX, Euro Stoxx 50), USA (S&P 500, Nasdaq 100, Dow), Wielka Brytania (FTSE 100), Polska (WIG20), Szwajcaria (SMI), Japonia (Nikkei 225), ropa WTI. Filtrujesz, co cię interesuje, i dostajesz przypomnienia.

## Pobieranie

- **Windows:** Releases → `windows` → `WydarzeniaGieldowe-Setup.exe` (instalator: menu Start, start z Windows, odinstalowanie) albo `WydarzeniaGieldowe.exe` (wersja przenośna)
- **Android:** Releases → `android` → `wydarzenia-gieldowe.apk`, podpisany tym samym kluczem co Dziennik FBW i Kalendarz Wydarzeń

Obie wersje budują się automatycznie po każdym pushu na `main`.

## Źródła

| Źródło | Klucz | Co daje | Zasięg |
|---|---|---|---|
| **Forex Factory** (feed Fair Economy) | brak | Wszystkie publikacje makro z wpływem (high/medium/low), prognozą i poprzednim odczytem, decyzje i wystąpienia bankierów, święta bankowe | bieżący tydzień (+ następny, jeśli feed go poda) |
| **Kalendarze banków centralnych** | wbudowane | Decyzje Fed, EBC, BoE i BoJ (z konferencjami Fed i EBC) | grudzień 2026 – grudzień 2027 |
| **Kalendarz NBP** | wbudowane | Decyzja RPP | grudzień 2026 (harmonogram na 2027 jeszcze nieopublikowany) |
| **Reguły giełd – wygasanie** | liczone | Eurex: FDAX, FESX i opcje, rollover. CME/Cboe: quad witching ES/NQ/YM, rollover, miesięczny OPEX, VIX. ICE: FTSE 100. GPW: FW20. JPX: Nikkei SQ (2. piątek). Ropa WTI (CL) | dowolnie daleko |
| **Reguły giełd – sesje** | liczone | Dni bez sesji na Xetra/Eurex, NYSE, LSE, GPW, SIX i w Tokio, skrócone sesje w USA i Londynie | dowolnie daleko |
| **Zmiana czasu** | liczone | Okresy, w których dane z USA wychodzą o 13:30, a Wall Street otwiera się o 14:30 | dowolnie daleko |
| **Nasdaq** | brak | Daty wyników spółek z USA z listy tickerów (domyślnie mega capy) | 45 dni |
| **ICS** | brak | Dowolny kalendarz online, np. kalendarze finansowe spółek z DAX ze stron IR | zależnie od kalendarza |
| **Ręcznie** | — | Pojedyncze wydarzenie | — |

Ograniczenia, które warto znać:
- Forex Factory podaje tylko bieżący tydzień. Dalej w przód widać tylko to, co liczone, i posiedzenia Fed/EBC. Feed blokuje zbyt częste pobieranie, więc apka odświeża go najwyżej co godzinę, a ręcznie najwyżej co 5 minut.
- Daty posiedzeń banków centralnych są wpisane w kodzie (`FOMC`, `ECB`, `BOE`, `BOJ`, `RPP` w `ui/index.html`). Nowe harmonogramy trzeba dopisywać: RPP na 2027, pozostałe banki na 2028. Daty BoE na 2027 są wstępne. BoJ i RPP nie mają stałej godziny decyzji, więc są wpisane jako wydarzenia całodniowe.
- Daty wyników bez klucza są tylko dla spółek z USA (Nasdaq). Dla DAX, FTSE, WIG20 itd.: link ICS z kalendarza finansowego spółki albo wpis ręczny.
- Święta i reguły wygasania są liczone z reguł, a jednorazowych wyjątków (np. dzień żałoby narodowej) giełdy ogłaszają osobno.

## Opisy wydarzeń

Kliknięcie wydarzenia pokazuje opis: co to jest, jak czytać odczyt względem prognozy i na co wpływa. Opisy obejmują:
- **Dane makro:** inflacja, rynek pracy, PMI/ISM, ifo/ZEW, PKB, sprzedaż, zapasy ropy i inne.
- **Banki centralne:** decyzje, konferencje, protokoły, wystąpienia.
- **Wygasanie i rollover:** w tym jak broker koryguje CFD przy przewinięciu.
- **Sesje:** święta, skrócone sesje, zmiana czasu.
- **Wyniki spółek:** z wagą w Nasdaq 100, S&P 500 albo DAX i przeliczeniem, ile ruch akcji daje dla indeksu.

W Ustawieniach jest też słownik pojęć. Wagi spółek są orientacyjne (`WEIGHTS`, `DAX_W` w `ui/index.html`) i warto je aktualizować co kwartał.

## Analiza rynku

Zakładka **Analiza** (Windows, Android) i rozwijany dolny panel widgetu („▾ Analiza rynku”):
- **EUWAX Sentiment (DAX):** −100…+100, nastawienie prywatnych graczy z produktów lewarowanych (Börse Stuttgart przez API ONVISTA), z przebiegiem dnia.
- **CNN Fear & Greed:** wynik 0–100, 7 składowych i historia (tydzień, miesiąc, rok).
- **VIX:** poziom i ocena (spokój / nerwowo / panika).
- **Trend SMA 33/144** na M5, M15, H1, H4, D1 dla wybranych instrumentów (domyślnie DAX, NQ, ES, YM, do tego FTSE, Nikkei, złoto, ropa, EURUSD) i sygnał zgodności interwałów.
- **Otwarte luki** z dziennych świec, z poziomem domknięcia i odległością od ceny.
- Opcjonalne powiadomienia: zgodność wszystkich interwałów, przekroczenie progu EUWAX.

Ceny z Yahoo Finance (z opóźnieniem, DAX jako indeks kasowy, bo Yahoo nie ma FDAX). Wszystkie trzy źródła są nieoficjalne i mogą się zmienić.

## Filtry i przypomnienia

- Filtry: rynki, kategorie, minimalny wpływ, waluty, wystąpienia bankierów, ukrywanie po słowach w tytule i ukrywanie pojedynczych wydarzeń. Przycisk **● High** w nagłówku szybko zawęża wszystko do wysokiego wpływu.
- Przypomnienia: osobne wyprzedzenie dla wysokiego, średniego i niskiego wpływu oraz dla decyzji banków. Dla wydarzeń całodniowych przypomnienie dzień wcześniej i/lub rano. Do tego poranne podsumowanie dnia.
- Godziny są pokazywane w czasie lokalnym telefonu lub komputera. Obok widać czas oryginalny publikacji (np. `14:30` i pod spodem `08:30 NY`), gdy różni się od lokalnego.
- Synchronizacja automatyczna co 15 min, 30 min, 1, 2, 4, 6, 12 lub 24 godziny albo wyłączona (Ustawienia). Na Androidzie działa w tle przy zamkniętej aplikacji (JobScheduler uruchamia ten sam kod UI bez okna, `SyncJob.java`). Na Windows działa, gdy program jest uruchomiony.
- **Android:** przypomnienia przez AlarmManager, przetrwają restart telefonu. Planowanych jest 150 najbliższych i odświeżają się przy każdym otwarciu apki.
- **Windows:** program działa w tle z ikoną przy zegarze. Zamknięcie okna (X) tylko je chowa, a „Zakończ” jest w menu pod prawym przyciskiem na ikonie. Po najechaniu na ikonę widać najbliższe ważne wydarzenie. Przy „Uruchamiaj z Windows” program startuje od razu w zasobniku.
- **Widget na pulpicie (Windows):** małe okienko z odliczaniem do najbliższego ważnego wydarzenia i listą na dziś i jutro. Leży pod oknami, więc nie zasłania wykresów. Przy przypomnieniu wychodzi na wierzch bez zabierania fokusu (OK chowa go z powrotem, samo po 3 minutach). Przesuwa się za pasek tytułu, rozmiar zmienia się za krawędzie i róg, a 📌 przypina go do wybranego rogu ekranu (trzyma się rogu także po zmianie rozdzielczości albo paska zadań). Pozycja i rozmiar są zapamiętywane. Włączanie: menu ikony przy zegarze albo Ustawienia.

## Architektura

```
ui/index.html        UI + cała logika: źródła, generatory (Eurex, święta, DST, Fed/EBC), filtry, plan przypomnień. Dane w localStorage.
main.go              Windows: okno WebView2 + lokalny serwer 127.0.0.1:47631
                       /api/http       pobieranie bez CORS
                       /api/schedule   przypomnienia → powiadomienia Windows
                       /api/open-ics   „Do kalendarza” (Outlook / Kalendarz Windows)
                       /api/save       eksport do Dokumenty\WydarzeniaGieldowe
                       /api/autostart  HKCU\...\Run
win_native.go        powiadomienia (toast przez PowerShell), autostart, jedna instancja
android/             WebView + most window.Android (http, schedule, toCalendar, saveFile, powiadomienia)
```

Nowe źródło sieciowe to nowa funkcja `syncXxx()` zapisująca wydarzenia `{key,title,start,end,allDay,cat,ccy,impact,info,src}` do `S.cache`.

---
made by PanRodla · all rights reserved
