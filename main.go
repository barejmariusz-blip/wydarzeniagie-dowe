//go:build windows

// Wydarzenia Giełdowe — wersja na Windows.
// Okno WebView2 z interfejsem ui/index.html (ten sam co na Androidzie) i lokalny serwer:
// pobieranie danych bez CORS, przypomnienia (powiadomienia Windows), zapis plików, autostart.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

//go:embed ui/index.html
var indexHTML []byte

const appName = "Wydarzenia Giełdowe"
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 WydarzeniaGieldowe/0.1"

func msgBox(text string) {
	u := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(appName)
	u.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10)
}

func documentsDir() string {
	if p, err := windows.KnownFolderPath(windows.FOLDERID_Documents, 0); err == nil && p != "" {
		return p
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Documents")
}

func openExternal(target string) {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
}

func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

var httpClient = &http.Client{Timeout: 25 * time.Second}

// GET bez ograniczeń CORS: {status, body}
func proxyGet(u string) (int, string) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/calendar, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,pl;q=0.8")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	return resp.StatusCode, string(b)
}

var safeName = regexp.MustCompile(`[^\w.\-]+`)

func serve(ln net.Listener) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/api/http", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
			jsonOut(w, map[string]any{"status": 0, "body": "zły adres"})
			return
		}
		st, body := proxyGet(u)
		jsonOut(w, map[string]any{"status": st, "body": body})
	})
	mux.HandleFunc("/api/schedule", func(w http.ResponseWriter, r *http.Request) {
		var list []alarm
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&list); err != nil {
			w.WriteHeader(400)
			jsonOut(w, map[string]string{"error": err.Error()})
			return
		}
		setAlarms(list)
		jsonOut(w, map[string]any{"ok": true, "count": len(list)})
	})
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		var a alarm
		json.NewDecoder(r.Body).Decode(&a)
		go toast(a.Title, a.Body)
		jsonOut(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		name := safeName.ReplaceAllString(r.URL.Query().Get("name"), "_")
		if name == "" {
			name = "eksport.json"
		}
		dir := filepath.Join(documentsDir(), "WydarzeniaGieldowe")
		b, err := io.ReadAll(io.LimitReader(r.Body, 50<<20))
		if err == nil {
			err = os.MkdirAll(dir, 0755)
		}
		p := filepath.Join(dir, name)
		if err == nil {
			err = os.WriteFile(p, b, 0644)
		}
		if err != nil {
			jsonOut(w, map[string]string{"error": err.Error()})
			return
		}
		exec.Command("explorer", "/select,", p).Start()
		jsonOut(w, map[string]string{"path": p})
	})
	// Otwiera plik .ics w domyślnym kalendarzu (Outlook / Kalendarz Windows)
	mux.HandleFunc("/api/open-ics", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		p := filepath.Join(os.TempDir(), fmt.Sprintf("wydarzenie-%d.ics", time.Now().UnixNano()))
		if err := os.WriteFile(p, b, 0644); err != nil {
			jsonOut(w, map[string]string{"error": err.Error()})
			return
		}
		openExternal(p)
		jsonOut(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/open-url", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			openExternal(u)
		}
		jsonOut(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc("/api/autostart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req struct{ On bool }
			json.NewDecoder(r.Body).Decode(&req)
			if err := setAutostart(req.On); err != nil {
				jsonOut(w, map[string]any{"error": err.Error()})
				return
			}
		}
		jsonOut(w, map[string]bool{"on": autostartOn()})
	})
	http.Serve(ln, mux)
}

func main() {
	// Jedna instancja: drugi start tylko podnosi okno pierwszej
	name, _ := syscall.UTF16PtrFromString("PanRodla.WydarzeniaGieldowe")
	if _, err := windows.CreateMutex(nil, false, name); err == windows.ERROR_ALREADY_EXISTS {
		focusExisting()
		return
	}

	ln, err := net.Listen("tcp", "127.0.0.1:47631")
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			msgBox("Nie mogę uruchomić lokalnego serwera: " + err.Error())
			return
		}
	}
	go serve(ln)
	go alarmLoop()

	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(documentsDir(), "WydarzeniaGieldowe")
	}
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		// Dane (localStorage) trzymane tutaj — stały port = te same dane przy każdym starcie
		DataPath: filepath.Join(local, "WydarzeniaGieldowe", "webview"),
		WindowOptions: webview2.WindowOptions{
			Title:  appName,
			Width:  1100,
			Height: 900,
			IconId: 1,
			Center: true,
		},
	})
	if w == nil {
		msgBox("Brakuje komponentu Microsoft Edge WebView2.\nZainstaluj go ze strony Microsoft (WebView2 Runtime) i uruchom program ponownie.")
		openExternal("https://developer.microsoft.com/microsoft-edge/webview2/")
		return
	}
	defer w.Destroy()
	if len(os.Args) > 1 && os.Args[1] == "--min" {
		minimize(uintptr(w.Window()))
	}
	w.Navigate(fmt.Sprintf("http://%s/", ln.Addr().String()))
	w.Run()
}
