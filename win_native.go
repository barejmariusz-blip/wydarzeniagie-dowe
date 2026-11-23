//go:build windows

package main

import (
	"encoding/base64"
	"os"
	"os/exec"
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
			toast(a.Title, a.Body)
		}
		time.Sleep(15 * time.Second)
	}
}

func xmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

// Powiadomienie Windows (toast) przez PowerShell — bez dodatkowych bibliotek.
func toast(title, body string) {
	xml := `<toast><visual><binding template="ToastGeneric"><text>` + xmlEsc(title) + `</text><text>` + xmlEsc(body) +
		`</text><text placement="attribution">Wydarzenia Giełdowe</text></binding></visual><audio src="ms-winsoundevent:Notification.Reminder"/></toast>`
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] > $null
$x = New-Object Windows.Data.Xml.Dom.XmlDocument
$x.LoadXml('` + strings.ReplaceAll(xml, "'", "''") + `')
$t = [Windows.UI.Notifications.ToastNotification]::new($x)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe').Show($t)`
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		b[2*i] = byte(c)
		b[2*i+1] = byte(c >> 8)
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-EncodedCommand", base64.StdEncoding.EncodeToString(b))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	if err := cmd.Run(); err != nil {
		msgBeep()
	}
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
