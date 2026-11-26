//go:build windows

// Ikona w zasobniku (przy zegarze), działanie w tle po zamknięciu okna i mini-widget na pulpicie.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"
)

var (
	shell32 = windows.NewLazySystemDLL("shell32.dll")
	kernel  = windows.NewLazySystemDLL("kernel32.dll")

	pShellNotifyIcon   = shell32.NewProc("Shell_NotifyIconW")
	pRegisterClassEx   = user32.NewProc("RegisterClassExW")
	pCreateWindowEx    = user32.NewProc("CreateWindowExW")
	pDefWindowProc     = user32.NewProc("DefWindowProcW")
	pCallWindowProc    = user32.NewProc("CallWindowProcW")
	pSetWindowLongPtr  = user32.NewProc("SetWindowLongPtrW")
	pGetWindowLongPtr  = user32.NewProc("GetWindowLongPtrW")
	pGetMessage        = user32.NewProc("GetMessageW")
	pTranslateMessage  = user32.NewProc("TranslateMessage")
	pDispatchMessage   = user32.NewProc("DispatchMessageW")
	pCreatePopupMenu   = user32.NewProc("CreatePopupMenu")
	pAppendMenu        = user32.NewProc("AppendMenuW")
	pTrackPopupMenu    = user32.NewProc("TrackPopupMenu")
	pDestroyMenu       = user32.NewProc("DestroyMenu")
	pGetCursorPos      = user32.NewProc("GetCursorPos")
	pSetForeground     = user32.NewProc("SetForegroundWindow")
	pShowWindow        = user32.NewProc("ShowWindow")
	pIsWindowVisible   = user32.NewProc("IsWindowVisible")
	pIsIconic          = user32.NewProc("IsIconic")
	pSetWindowPos      = user32.NewProc("SetWindowPos")
	pGetWindowRect     = user32.NewProc("GetWindowRect")
	pReleaseCapture    = user32.NewProc("ReleaseCapture")
	pSendMessage       = user32.NewProc("SendMessageW")
	pPostMessage       = user32.NewProc("PostMessageW")
	pLoadImage         = user32.NewProc("LoadImageW")
	pGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	pSystemParamsInfo  = user32.NewProc("SystemParametersInfoW")
	pRegisterWindowMsg = user32.NewProc("RegisterWindowMessageW")
	pGetModuleHandle   = kernel.NewProc("GetModuleHandleW")
	pMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfo    = user32.NewProc("GetMonitorInfoW")
)

const (
	wmClose             = 0x0010
	wmWindowPosChanging = 0x0046
	wmExitSizeMove      = 0x0232
	wmNcLButtonDown     = 0x00A1
	wmLButtonUp         = 0x0202
	wmLButtonDblClk     = 0x0203
	wmRButtonUp         = 0x0205
	wmCommand           = 0x0111
	wmApp               = 0x8000
	wmTray              = wmApp + 11
	htCaption           = 2
	swHide              = 0
	swShow              = 5
	swRestore           = 9
	swShowNoActivate    = 4
	gwlWndProc          = -4
	gwlStyle            = -16
	gwlExStyle          = -20
	wsPopup             = 0x80000000
	wsVisible           = 0x10000000
	wsClipChildren      = 0x02000000
	wsExToolWindow      = 0x00000080
	wsExNoActivate      = 0x08000000
	swpNoSize           = 0x0001
	swpNoMove           = 0x0002
	swpNoZOrder         = 0x0004
	swpNoActivate       = 0x0010
	swpFrameChanged     = 0x0020
	swpShowWindow       = 0x0040
	hwndTop             = 0
	hwndBottom          = 1
	hwndTopmost         = ^uintptr(0)     // -1
	hwndNoTopmost       = ^uintptr(0) - 1 // -2
	nimAdd              = 0
	nimModify           = 1
	nimDelete           = 2
	nimSetVersion       = 4
	nifMessage          = 0x01
	nifIcon             = 0x02
	nifTip              = 0x04
	nifInfo             = 0x10
	nifShowTip          = 0x80
	niifInfo            = 0x01
	mfString            = 0x0000
	mfChecked           = 0x0008
	mfSeparator         = 0x0800
	tpmRightButton      = 0x0002
	tpmReturnCmd        = 0x0100
	tpmNoNotify         = 0x0080
	idOpen              = 1
	idWidget            = 2
	idSync              = 3
	idQuit              = 9
	widgetW             = 320
	widgetH             = 400
)

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     uintptr
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type winMsg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

type rect struct{ Left, Top, Right, Bottom int32 }

type windowPos struct {
	HWnd            uintptr
	HWndInsertAfter uintptr
	X, Y, Cx, Cy    int32
	Flags           uint32
}

// ---------------------------------------------------------------- konfiguracja (widget)

type appConfig struct {
	Widget  bool   `json:"widget"`
	WX      int32  `json:"wx"`
	WY      int32  `json:"wy"`
	HasPos  bool   `json:"hasPos"`
	Balloon bool   `json:"balloon"` // pokazano już dymek „działa w tle”
	WW      int32  `json:"ww"`      // rozmiar widgetu
	WH      int32  `json:"wh"`
	Corner  string `json:"corner"` // "" = swobodnie, "tl" | "tr" | "bl" | "br" = przypięty do rogu ekranu
}

var (
	cfgMu sync.Mutex
	cfg   = appConfig{Widget: true}
)

func cfgPath() string {
	d := os.Getenv("LOCALAPPDATA")
	if d == "" {
		d = documentsDir()
	}
	return filepath.Join(d, "WydarzeniaGieldowe", "config.json")
}

func loadConfig() {
	if b, err := os.ReadFile(cfgPath()); err == nil {
		json.Unmarshal(b, &cfg)
	}
}

func saveConfig() {
	cfgMu.Lock()
	b, _ := json.MarshalIndent(cfg, "", "  ")
	cfgMu.Unlock()
	os.MkdirAll(filepath.Dir(cfgPath()), 0755)
	os.WriteFile(cfgPath(), b, 0644)
}

// ---------------------------------------------------------------- stan okien

var (
	mainWV      webview2.WebView
	mainHwnd    uintptr
	mainOldProc uintptr
	quitting    bool

	widgetWV      webview2.WebView
	widgetHwnd    uintptr
	widgetOldProc uintptr
	baseURL       string

	alertMu     sync.Mutex
	widgetAlert *alarm
	alertOn     bool
	alertSince  time.Time

	trayHwnd uintptr
	trayTip  = appName
	tipMu    sync.Mutex
)

func hinstance() uintptr { h, _, _ := pGetModuleHandle.Call(0); return h }

func appIcon(small bool) uintptr {
	sz := uintptr(0)
	if small {
		s, _, _ := pGetSystemMetrics.Call(49) // SM_CXSMICON
		sz = s
	}
	h, _, _ := pLoadImage.Call(hinstance(), 1, 1 /*IMAGE_ICON*/, sz, sz, 0)
	if h == 0 {
		h, _, _ = pLoadImage.Call(0, 32512, 1, sz, sz, 0x8000 /*LR_SHARED*/)
	}
	return h
}

func showMain() {
	if mainHwnd == 0 {
		return
	}
	if iconic, _, _ := pIsIconic.Call(mainHwnd); iconic != 0 {
		pShowWindow.Call(mainHwnd, swRestore)
	} else {
		pShowWindow.Call(mainHwnd, swShow)
	}
	pSetForeground.Call(mainHwnd)
}

// Zamknięcie okna głównego (X) chowa je do zasobnika zamiast kończyć program.
func mainProc(hwnd, msg, wp, lp uintptr) uintptr {
	if msg == wmClose && !quitting {
		pShowWindow.Call(hwnd, swHide)
		cfgMu.Lock()
		first := !cfg.Balloon
		cfg.Balloon = true
		cfgMu.Unlock()
		if first {
			saveConfig()
			balloon("Program działa w tle", "Przypomnienia i synchronizacja działają dalej. Ikona jest przy zegarze, a „Zakończ” jest pod prawym przyciskiem myszy.")
		}
		return 0
	}
	r, _, _ := pCallWindowProc.Call(mainOldProc, hwnd, msg, wp, lp)
	return r
}

func quitApp() {
	if mainWV == nil {
		os.Exit(0)
	}
	mainWV.Dispatch(func() {
		quitting = true
		removeTray()
		mainWV.Terminate()
	})
}

// ---------------------------------------------------------------- widget

// Widget leży pod oknami (nie zasłania wykresów). Przy przypomnieniu wychodzi na wierzch bez zabierania fokusu.
func widgetProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmClose:
		setWidget(false)
		return 0
	case wmWindowPosChanging:
		alertMu.Lock()
		on := alertOn
		alertMu.Unlock()
		if !on {
			p := *(**windowPos)(unsafe.Pointer(&lp)) // lParam = WINDOWPOS*
			p.HWndInsertAfter = hwndBottom
			p.Flags &^= swpNoZOrder
		}
	case 0x0083: // WM_NCCALCSIZE: bez widocznej ramki (WS_THICKFRAME tylko dla zmiany rozmiaru)
		if wp != 0 {
			return 0
		}
	case 0x0024: // WM_GETMINMAXINFO: minimalny rozmiar
		mm := *(**[10]int32)(unsafe.Pointer(&lp))
		mm[6], mm[7] = minWidgetW, minWidgetH
		return 0
	case wmExitSizeMove:
		var r rect
		pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
		cfgMu.Lock()
		if dragging {
			cfg.Corner = "" // przeciągnięty ręcznie: odpięty od rogu
		}
		cfg.WX, cfg.WY, cfg.HasPos = r.Left, r.Top, true
		cfg.WW, cfg.WH = r.Right-r.Left, r.Bottom-r.Top
		cfgMu.Unlock()
		dragging = false
		saveConfig()
		anchorWidget()
	case 0x007E, 0x001A: // WM_DISPLAYCHANGE, WM_SETTINGCHANGE (rozdzielczość, pasek zadań)
		go func() { time.Sleep(500 * time.Millisecond); anchorWidget() }()
	}
	r, _, _ := pCallWindowProc.Call(widgetOldProc, hwnd, msg, wp, lp)
	return r
}

// Musi być wołane w wątku okien (przed Run albo przez Dispatch).
func createWidget() {
	if widgetWV != nil {
		pShowWindow.Call(widgetHwnd, swShowNoActivate)
		pSetWindowPos.Call(widgetHwnd, hwndBottom, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		return
	}
	local := os.Getenv("LOCALAPPDATA")
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath:      filepath.Join(local, "WydarzeniaGieldowe", "webview"),
		WindowOptions: webview2.WindowOptions{Title: appName + " – widget", Width: widgetW, Height: widgetH, IconId: 1},
	})
	if wv == nil {
		return
	}
	widgetWV = wv
	widgetHwnd = uintptr(wv.Window())
	// bez ramki, bez paska zadań i Alt+Tab
	pSetWindowLongPtr.Call(widgetHwnd, negIdx(gwlStyle), wsPopup|wsVisible|wsClipChildren)
	ex, _, _ := pGetWindowLongPtr.Call(widgetHwnd, negIdx(gwlExStyle))
	pSetWindowLongPtr.Call(widgetHwnd, negIdx(gwlExStyle), ex|wsExToolWindow)
	widgetOldProc, _, _ = pSetWindowLongPtr.Call(widgetHwnd, negIdx(gwlWndProc), windows.NewCallback(widgetProc))

	cfgMu.Lock()
	x, y, has, ww, wh := cfg.WX, cfg.WY, cfg.HasPos, cfg.WW, cfg.WH
	if !has {
		cfg.Corner = "br" // pierwszy start: prawy dolny róg
	}
	cfgMu.Unlock()
	if ww < minWidgetW || wh < minWidgetH {
		ww, wh = widgetW, widgetH
	}
	// WS_THICKFRAME pozwala zmieniać rozmiar; ramkę ukrywa WM_NCCALCSIZE
	pSetWindowLongPtr.Call(widgetHwnd, negIdx(gwlStyle), wsPopup|wsVisible|wsClipChildren|0x00040000)
	pSetWindowPos.Call(widgetHwnd, hwndBottom, uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), swpNoActivate|swpFrameChanged|swpShowWindow)
	anchorWidget()

	// Przeciąganie musi ruszyć w wątku okien w trakcie wciśniętego przycisku: Bind działa synchronicznie.
	// (Pozostałe akcje widgetu idą przez /api/…)
	wv.Bind("wgDrag", func() {
		dragging = true
		pReleaseCapture.Call()
		pSendMessage.Call(widgetHwnd, wmNcLButtonDown, htCaption, 0)
	})
	// Zmiana rozmiaru za krawędź: ht = HTLEFT(10)…HTBOTTOMRIGHT(17)
	wv.Bind("wgResize", func(ht int) {
		if ht < 10 || ht > 17 {
			return
		}
		pReleaseCapture.Call()
		pSendMessage.Call(widgetHwnd, wmNcLButtonDown, uintptr(ht), 0)
	})
	wv.Navigate(baseURL + "?widget=1")
}

var dragging bool

const (
	minWidgetW = 220
	minWidgetH = 160
	pinMargin  = 12
)

type monitorInfo struct {
	CbSize  uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

// Przypięty widget trzyma się wybranego rogu obszaru roboczego monitora (bez paska zadań).
func anchorWidget() {
	if widgetHwnd == 0 {
		return
	}
	cfgMu.Lock()
	c := cfg.Corner
	cfgMu.Unlock()
	if c == "" {
		return
	}
	var r rect
	pGetWindowRect.Call(widgetHwnd, uintptr(unsafe.Pointer(&r)))
	w, h := r.Right-r.Left, r.Bottom-r.Top
	mon, _, _ := pMonitorFromWindow.Call(widgetHwnd, 2 /*MONITOR_DEFAULTTONEAREST*/)
	mi := monitorInfo{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if ok, _, _ := pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		pSystemParamsInfo.Call(0x0030, 0, uintptr(unsafe.Pointer(&mi.Work)), 0)
	}
	wa := mi.Work
	x, y := wa.Left+pinMargin, wa.Top+pinMargin
	if c == "tr" || c == "br" {
		x = wa.Right - w - pinMargin
	}
	if c == "bl" || c == "br" {
		y = wa.Bottom - h - pinMargin
	}
	if x == r.Left && y == r.Top {
		return
	}
	pSetWindowPos.Call(widgetHwnd, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
	cfgMu.Lock()
	cfg.WX, cfg.WY, cfg.HasPos = x, y, true
	cfgMu.Unlock()
	saveConfig()
}

func setCorner(c string) {
	switch c {
	case "", "tl", "tr", "bl", "br":
	default:
		return
	}
	cfgMu.Lock()
	cfg.Corner = c
	cfgMu.Unlock()
	saveConfig()
	anchorWidget()
}

func widgetCorner() string { cfgMu.Lock(); defer cfgMu.Unlock(); return cfg.Corner }

func negIdx(i int) uintptr { return uintptr(i) } // indeksy GWL są ujemne: konwersja z zachowaniem bitów

func widgetVisible() bool {
	if widgetHwnd == 0 {
		return false
	}
	v, _, _ := pIsWindowVisible.Call(widgetHwnd)
	return v != 0
}

func setWidget(on bool) {
	cfgMu.Lock()
	cfg.Widget = on
	cfgMu.Unlock()
	saveConfig()
	if on {
		if mainWV != nil {
			mainWV.Dispatch(func() {
				createWidget()
				mainWV.Dispatch(func() {}) // ponowne „obudzenie” kolejki po wewnętrznej pętli tworzenia okna
			})
		}
	} else if widgetHwnd != 0 {
		pShowWindow.Call(widgetHwnd, swHide)
	}
}

func widgetEnabled() bool { cfgMu.Lock(); defer cfgMu.Unlock(); return cfg.Widget }

func raiseWidget(a alarm) {
	if widgetHwnd == 0 || !widgetEnabled() {
		return
	}
	alertMu.Lock()
	widgetAlert = &a
	alertOn = true
	alertSince = time.Now()
	alertMu.Unlock()
	pShowWindow.Call(widgetHwnd, swShowNoActivate)
	pSetWindowPos.Call(widgetHwnd, hwndTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
}

func ackAlert() {
	alertMu.Lock()
	widgetAlert = nil
	alertOn = false
	alertMu.Unlock()
	if widgetHwnd != 0 {
		pSetWindowPos.Call(widgetHwnd, hwndNoTopmost, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		pSetWindowPos.Call(widgetHwnd, hwndBottom, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
	}
}

// Po 3 minutach bez reakcji widget sam wraca pod okna.
func alertTimeoutLoop() {
	for {
		time.Sleep(10 * time.Second)
		alertMu.Lock()
		stale := alertOn && time.Since(alertSince) > 3*time.Minute
		alertMu.Unlock()
		if stale {
			ackAlert()
		}
	}
}

func currentAlert() any {
	alertMu.Lock()
	defer alertMu.Unlock()
	if widgetAlert == nil {
		return nil
	}
	return widgetAlert
}

// ---------------------------------------------------------------- zasobnik

var taskbarCreated uintptr

func trayData() notifyIconData {
	nid := notifyIconData{UID: 1, HWnd: trayHwnd}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	return nid
}

func addTray() {
	nid := trayData()
	nid.UFlags = nifMessage | nifIcon | nifTip | nifShowTip
	nid.UCallbackMessage = wmTray
	nid.HIcon = appIcon(true)
	tipMu.Lock()
	copy(nid.SzTip[:127], windows.StringToUTF16(trayTip))
	tipMu.Unlock()
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	nid.UVersion = 4
	pShellNotifyIcon.Call(nimSetVersion, uintptr(unsafe.Pointer(&nid)))
}

func removeTray() {
	if trayHwnd == 0 {
		return
	}
	nid := trayData()
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func setTrayTip(t string) {
	if t == "" {
		t = appName
	}
	tipMu.Lock()
	trayTip = t
	tipMu.Unlock()
	if trayHwnd == 0 {
		return
	}
	nid := trayData()
	nid.UFlags = nifTip | nifShowTip
	copy(nid.SzTip[:127], windows.StringToUTF16(t))
	pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func balloon(title, text string) {
	if trayHwnd == 0 {
		return
	}
	nid := trayData()
	nid.UFlags = nifInfo
	nid.DwInfoFlags = niifInfo
	copy(nid.SzInfoTitle[:63], windows.StringToUTF16(title))
	copy(nid.SzInfo[:255], windows.StringToUTF16(text))
	pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func trayMenu(hwnd uintptr) {
	m, _, _ := pCreatePopupMenu.Call()
	add := func(flags uintptr, id uintptr, text string) {
		p, _ := windows.UTF16PtrFromString(text)
		pAppendMenu.Call(m, flags, id, uintptr(unsafe.Pointer(p)))
	}
	add(mfString, idOpen, "Otwórz")
	wf := uintptr(mfString)
	if widgetEnabled() {
		wf |= mfChecked
	}
	add(wf, idWidget, "Widget na pulpicie")
	add(mfString, idSync, "Odśwież teraz")
	add(mfSeparator, 0, "")
	add(mfString, idQuit, "Zakończ")
	var pt struct{ X, Y int32 }
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForeground.Call(hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(m, tpmRightButton|tpmReturnCmd|tpmNoNotify, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	pDestroyMenu.Call(m)
	switch cmd {
	case idOpen:
		showMain()
	case idWidget:
		setWidget(!widgetEnabled())
	case idSync:
		if mainWV != nil {
			mainWV.Dispatch(func() { mainWV.Eval("window.syncAll&&syncAll(true)") })
		}
	case idQuit:
		quitApp()
	}
}

func trayProc(hwnd, msg, wp, lp uintptr) uintptr {
	if msg == wmTray {
		switch uint32(lp) & 0xFFFF {
		case wmLButtonUp, wmLButtonDblClk:
			showMain()
		case wmRButtonUp, 0x007B /*WM_CONTEXTMENU*/ :
			trayMenu(hwnd)
		}
		return 0
	}
	if taskbarCreated != 0 && msg == taskbarCreated { // Eksplorator uruchomiony ponownie
		addTray()
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

// Własny wątek z pętlą komunikatów dla ikony w zasobniku.
func trayLoop() {
	runtime.LockOSThread()
	cls, _ := windows.UTF16PtrFromString("WydarzeniaGieldoweTray")
	wc := wndClassEx{LpfnWndProc: windows.NewCallback(trayProc), HInstance: hinstance(), LpszClassName: cls}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	tc, _ := windows.UTF16PtrFromString("TaskbarCreated")
	taskbarCreated, _, _ = pRegisterWindowMsg.Call(uintptr(unsafe.Pointer(tc)))
	trayHwnd, _, _ = pCreateWindowEx.Call(0, uintptr(unsafe.Pointer(cls)), 0, 0, 0, 0, 0, 0, 0, 0, hinstance(), 0)
	addTray()
	var m winMsg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || int32(r) == -1 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// Wywoływane z main po utworzeniu okna głównego.
func setupShell(w webview2.WebView, url string, startHidden bool) {
	mainWV = w
	mainHwnd = uintptr(w.Window())
	baseURL = url
	mainOldProc, _, _ = pSetWindowLongPtr.Call(mainHwnd, negIdx(gwlWndProc), windows.NewCallback(mainProc))
	if startHidden {
		pShowWindow.Call(mainHwnd, swHide)
	}
	go trayLoop()
	go alertTimeoutLoop()
	if widgetEnabled() {
		createWidget()
	}
}
