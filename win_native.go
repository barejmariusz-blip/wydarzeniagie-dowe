//go:build windows

package main

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ---------------------------------------------------------------- przypomnienia

type alarm struct {
	At    int64  `json:"at"` // ms od 1970
	Title string `json:"title"`
	Body  string `json:"body"`
}

var (
	amu    sync.Mutex
	alarms []alarm
	fired  = map[string]bool{}
)

func alarmKey(a alarm) string { return a.Title + "|" + time.UnixMilli(a.At).Format(time.RFC3339) }

func setAlarms(list []alarm) {
	amu.Lock()
	alarms = list
	amu.Unlock()
}

// Co 15 s sprawdza, czy coś jest do pokazania. Spóźnienie do 10 min (np. po uśpieniu) nadal pokazuje.
func alarmLoop() {
	for {
		now := time.Now().UnixMilli()
		var due []alarm
		amu.Lock()
		for _, a := range alarms {
			k := alarmKey(a)
			if a.At <= now && now-a.At < 10*60*1000 && !fired[k] {
				fired[k] = true
				due = append(due, a)
			}
		}
		amu.Unlock()
		for _, a := range due {
			raiseWidget(a)
			toast(a.Title, a.Body)
			notifyUI(a)
		}
		time.Sleep(15 * time.Second)
	}
}

func xmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

//go:embed assets/icon.png
var iconPNG []byte

const appID = "PanRodla.WydarzeniaGieldowe"
const psAppID = `{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe`

// Rejestruje własny identyfikator aplikacji dla powiadomień (bez instalatora):
// HKCU\Software\Classes\AppUserModelId\<id> z nazwą i ikoną. Dzięki temu toast ma nazwę
// „Wydarzenia Giełdowe” i własny przełącznik w Ustawienia → System → Powiadomienia.
func registerAppID() {
	icon := ""
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		p := filepath.Join(dir, "WydarzeniaGieldowe", "icon.png")
		if os.MkdirAll(filepath.Dir(p), 0755) == nil && os.WriteFile(p, iconPNG, 0644) == nil {
			icon = p
		}
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\AppUserModelId\`+appID, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	k.SetStringValue("DisplayName", appName)
	if icon != "" {
		k.SetStringValue("IconUri", icon)
	}
	k.SetStringValue("IconBackgroundColor", "FF1A1F2B")
	k.SetDWordValue("ShowInSettings", 1)
}

func runPS(script string) (string, error) {
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i] = byte(c)
		b[2*i+1] = byte(c >> 8)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Powiadomienie Windows (toast) przez PowerShell — bez dodatkowych bibliotek.
// Zwraca stan powiadomień dla aplikacji ("Enabled", "DisabledForUser"…) albo błąd.
func toast(title, body string) (string, error) {
	xml := `<toast scenario="reminder"><visual><binding template="ToastGeneric"><text>` + xmlEsc(title) + `</text><text>` + xmlEsc(body) +
		`</text></binding></visual><actions><action content="OK" arguments="dismiss" activationType="system"/></actions>` +
		`<audio src="ms-winsoundevent:Notification.Reminder"/></toast>`
	try := func(id string) (string, error) {
		script := `$ErrorActionPreference = 'Stop'
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null
$x = New-Object Windows.Data.Xml.Dom.XmlDocument
$x.LoadXml('` + strings.ReplaceAll(xml, "'", "''") + `')
$t = [Windows.UI.Notifications.ToastNotification]::new($x)
$n = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('` + id + `')
$n.Show($t)
Write-Output $n.Setting`
		out, err := runPS(script)
		if err != nil {
			if out == "" {
				out = err.Error()
			}
			return "", fmt.Errorf("%s", out)
		}
		lines := strings.Split(out, "\n")
		return strings.TrimSpace(lines[len(lines)-1]), nil
	}
	st, err := try(appID)
	if err != nil || st != "Enabled" {
		// zapasowo: identyfikator Windows PowerShell (zawsze zarejestrowany)
		if st2, err2 := try(psAppID); err2 == nil && st2 == "Enabled" {
			return st2, nil
		} else if err == nil {
			return st, nil
		}
	}
	if err != nil {
		msgBeep()
	}
	return st, err
}

func msgBeep() { syscall.NewLazyDLL("user32.dll").NewProc("MessageBeep").Call(0x40) }

// ---------------------------------------------------------------- autostart (HKCU\...\Run)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const runName = "WydarzeniaGieldowe"

func autostartOn() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runName)
	return err == nil
}

func setAutostart(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		k.DeleteValue(runName)
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runName, `"`+exe+`" --min`)
}

// ---------------------------------------------------------------- okno

var user32 = windows.NewLazySystemDLL("user32.dll")

func minimize(hwnd uintptr) { user32.NewProc("ShowWindow").Call(hwnd, 6) } // SW_MINIMIZE

func focusExisting() {
	t, _ := syscall.UTF16PtrFromString(appName)
	h, _, _ := user32.NewProc("FindWindowW").Call(0, uintptr(unsafe.Pointer(t)))
	if h != 0 {
		user32.NewProc("ShowWindow").Call(h, 9) // SW_RESTORE
		user32.NewProc("SetForegroundWindow").Call(h)
	}
}

var (
	fmu       sync.Mutex
	firedList []alarm
)

func notifyUI(a alarm) {
	fmu.Lock()
	firedList = append(firedList, a)
	if len(firedList) > 20 {
		firedList = firedList[len(firedList)-20:]
	}
	fmu.Unlock()
}

func takeFired() []alarm {
	fmu.Lock()
	defer fmu.Unlock()
	out := firedList
	firedList = nil
	if out == nil {
		out = []alarm{}
	}
	return out
}
