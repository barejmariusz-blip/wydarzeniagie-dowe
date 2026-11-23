# Wydarzenia Giełdowe

Agregator wydarzeń rynkowych dla tradera DAX na Androida i Windows. Ten sam interfejs w obu wersjach (`ui/index.html`).

Zbiera wszystko w jeden kalendarz: dane makro, decyzje banków centralnych, wygasanie FDAX i opcji, rollover, święta giełdowe, skrócone sesje, zmianę czasu USA/Europa i wyniki spółek. Filtrujesz, co cię interesuje, i dostajesz przypomnienia.

## Pobieranie

- **Windows:** Releases → `windows` → `WydarzeniaGieldowe.exe`
- **Android:** Releases → `android` → `wydarzenia-gieldowe.apk`, podpisany tym samym kluczem co Dziennik FBW i Kalendarz Wydarzeń

Obie wersje budują się automatycznie po każdym pushu na `main`.

## Źródła

| Źródło | Klucz | Co daje | Zasięg |
|---|---|---|---|
| **Forex Factory** (feed Fair Economy) | brak | Wszystkie publikacje makro z wpływem (high/medium/low), prognozą i poprzednim odczytem, decyzje i wystąpienia bankierów, święta bankowe | bieżący tydzień (+ następny, jeśli feed go poda) |
| **Kalendarze Fed i EBC** | wbudowane | Decyzje i konferencje FOMC i EBC | grudzień 2026 – grudzień 2027 |
| **Reguły Eurex** | liczone | Wygasanie FDAX i opcji na DAX (3. piątek, 13:00, kwartały oznaczone jako high), rollover ok. tydzień przed | dowolnie daleko |
| **Reguły giełd** | liczone | Xetra/Eurex zamknięte, święta i skrócone sesje NYSE | dowolnie daleko |
| **Zmiana czasu** | liczone | Okresy, w których dane z USA wychodzą o 13:30, a Wall Street otwiera się o 14:30 | dowolnie daleko |
| **Nasdaq** | brak | Daty wyników spółek z USA z listy tickerów (domyślnie mega capy) | 45 dni |
| **ICS** | brak | Dowolny kalendarz online, np. kalendarze finansowe spółek z DAX ze stron IR | zależnie od kalendarza |
| **Ręcznie** | — | Pojedyncze wydarzenie | — |

Ograniczenia, które warto znać:
- Forex Factory podaje tylko bieżący tydzień. Dalej w przód widać tylko to, co liczone, i posiedzenia Fed/EBC. Feed blokuje zbyt częste pobieranie, więc apka odświeża go najwyżej co godzinę, a ręcznie najwyżej co 5 minut.
- Daty posiedzeń Fed i EBC są wpisane w kodzie (`FOMC`, `ECB` w `ui/index.html`). Gdy banki opublikują kalendarz na 2028, trzeba je dopisać.
- Darmowego źródła dat wyników dla całego DAX bez klucza nie ma. Rozwiązanie: link ICS z kalendarza finansowego danej spółki albo wpis ręczny.

## Filtry i przypomnienia

- Filtry: kategorie, minimalny wpływ, waluty, wystąpienia bankierów, ukrywanie po słowach w tytule i ukrywanie pojedynczych wydarzeń. Przycisk **● High** w nagłówku szybko zawęża wszystko do wysokiego wpływu.
- Przypomnienia: osobne wyprzedzenie dla wysokiego, średniego i niskiego wpływu oraz dla decyzji banków. Dla wydarzeń całodniowych przypomnienie dzień wcześniej i/lub rano. Do tego poranne podsumowanie dnia.
- Wszystkie godziny są pokazywane w czasie lokalnym telefonu lub komputera.
- **Android:** przypomnienia przez AlarmManager, przetrwają restart telefonu. Planowanych jest 150 najbliższych i odświeżają się przy każdym otwarciu apki.
- **Windows:** powiadomienia systemowe działają, gdy program jest uruchomiony (może być zminimalizowany). W Ustawieniach można włączyć „Uruchamiaj z Windows”: program startuje wtedy zminimalizowany.

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
