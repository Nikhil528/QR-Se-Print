//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const defaultServer = "https://bvv-djql.onrender.com"
const appVersion = "2.7"

type Config struct {
	ServerURL     string `json:"ServerURL"`
	ShopID        string `json:"ShopID"`
	AgentToken    string `json:"AgentToken"`
	AgentID       string `json:"AgentID"`
	PrinterBW     string `json:"PrinterBW"`
	PrinterColor  string `json:"PrinterColor"`
	Printer4x6    string `json:"Printer4x6"`
	PrinterA3     string `json:"PrinterA3"`
	PrinterDuplex string `json:"PrinterDuplex"`
	BWPrinter     string `json:"BWPrinter"`
	ColorPrinter  string `json:"ColorPrinter"`
	PollSeconds   int    `json:"PollSeconds"`
	Version       string `json:"Version"`
	SumatraPath   string `json:"SumatraPath"`
}
type Job struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	FileURL       string `json:"file_url"`
	FileName      string `json:"file_name"`
	Copies        int    `json:"copies"`
	ColorMode     string `json:"color_mode"`
	Service       string `json:"service"`
	PaperSize     string `json:"paper_size"`
	Duplex        bool   `json:"duplex"`
	PaymentMethod string `json:"payment_method"`
	PaymentStatus string `json:"payment_status"`
}
type APIJobResponse struct {
	Success bool   `json:"success"`
	Job     *Job   `json:"job"`
	Jobs    []Job  `json:"jobs"`
	Error   string `json:"error"`
}
type LoginResponse struct {
	Success     bool   `json:"success"`
	ShopID      string `json:"shopId"`
	AgentToken  string `json:"agentToken"`
	AgentID     string `json:"agentId"`
	PollSeconds int    `json:"pollSeconds"`
	Version     string `json:"version"`
	ShopName    string `json:"shopName"`
	Error       string `json:"error"`
}
type Printer struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

// ---------------- Win32 ----------------
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	shell32              = syscall.NewLazyDLL("shell32.dll")
	winspool             = syscall.NewLazyDLL("winspool.drv")
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegOpen          = advapi32.NewProc("RegOpenKeyExW")
	procRegSet           = advapi32.NewProc("RegSetValueExW")
	procRegClose         = advapi32.NewProc("RegCloseKey")
	procRegisterClass    = user32.NewProc("RegisterClassW")
	procCreateWindow     = user32.NewProc("CreateWindowExW")
	procDefWindow        = user32.NewProc("DefWindowProcW")
	procShowWindow       = user32.NewProc("ShowWindow")
	procUpdateWindow     = user32.NewProc("UpdateWindow")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslate        = user32.NewProc("TranslateMessage")
	procDispatch         = user32.NewProc("DispatchMessageW")
	procPostQuit         = user32.NewProc("PostQuitMessage")
	procDestroy          = user32.NewProc("DestroyWindow")
	procSetWindowText    = user32.NewProc("SetWindowTextW")
	procGetWindowText    = user32.NewProc("GetWindowTextW")
	procGetClientRect    = user32.NewProc("GetClientRect")
	procInvalidate       = user32.NewProc("InvalidateRect")
	procMessageBox       = user32.NewProc("MessageBoxW")
	procSendMessage      = user32.NewProc("SendMessageW")
	procEnableWindow     = user32.NewProc("EnableWindow")
	procSetFocus         = user32.NewProc("SetFocus")
	procGetSysColorBrush = user32.NewProc("GetSysColorBrush")
	procLoadCursor       = user32.NewProc("LoadCursorW")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	procShellExecute     = shell32.NewProc("ShellExecuteW")
	procGetModule        = kernel32.NewProc("GetModuleHandleW")
	procGetLastError     = kernel32.NewProc("GetLastError")
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procFillRect         = user32.NewProc("FillRect")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procCreateFont       = gdi32.NewProc("CreateFontW")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procTextOut          = user32.NewProc("DrawTextW")
	procEnumPrinters     = winspool.NewProc("EnumPrintersW")
	procShellNotify      = shell32.NewProc("Shell_NotifyIconW")
)

const (
	HKEY_CURRENT_USER   = 0x80000001
	KEY_SET_VALUE       = 0x0002
	REG_SZ              = 1
	WM_DRAWITEM         = 43
	BS_OWNERDRAW        = 0x0000000B
	WM_CREATE           = 1
	WM_DESTROY          = 2
	WM_PAINT            = 15
	WM_COMMAND          = 273
	WM_CLOSE            = 16
	WM_CTLCOLORSTATIC   = 312
	WM_CTLCOLOREDIT     = 307
	WM_TIMER            = 275
	WM_USER             = 1024
	WM_LBUTTONDBLCLK    = 515
	WM_RBUTTONUP        = 517
	SW_SHOW             = 5
	SW_HIDE             = 0
	SW_SHOWNORMAL       = 1
	CW_USEDEFAULT       = 0x80000000
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_TABSTOP          = 0x00010000
	ES_PASSWORD         = 0x20
	ES_AUTOHSCROLL      = 0x80
	BS_PUSHBUTTON       = 0
	BS_DEFPUSHBUTTON    = 1
	CBS_DROPDOWNLIST    = 3
	WS_EX_APPWINDOW     = 0x40000
	WS_EX_CLIENTEDGE    = 0x200
	COLOR_WINDOW        = 5
	COLOR_BTNFACE       = 15
	GWLP_USERDATA       = -21
	WM_SETFONT          = 48
	CB_ADDSTRING        = 323
	CB_RESETCONTENT     = 331
	CB_SETCURSEL        = 334
	CB_GETCURSEL        = 327
	CB_GETLBTEXTLEN     = 329
	CB_GETLBTEXT        = 328
	BN_CLICKED          = 0
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type MSG struct {
	HWnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
}
type WNDCLASS struct {
	Style                                    uint32
	WndProc                                  uintptr
	ClsExtra, WndExtra                       int32
	HInstance, HIcon, HCursor, HbrBackground uintptr
	MenuName, ClassName                      *uint16
}
type PAINTSTRUCT struct {
	Hdc                uintptr
	Erase              uint32
	RcPaint            RECT
	Restore, IncUpdate int32
	RgbReserved        [32]byte
}
type PRINTER_INFO_4 struct {
	PPrinterName *uint16
	PServerName  *uint16
	Attributes   uint32
}
type DRAWITEMSTRUCT struct {
	CtlType, CtlID, ItemAction, ItemState uint32
	HWndItem, HDC                         uintptr
	RcItem                                RECT
	ItemData                              uintptr
}
type NOTIFYICONDATA struct {
	CbSize                    uint32
	HWnd                      uintptr
	UID                       uint32
	Flags                     uint32
	Callback                  uint32
	HIcon                     uintptr
	Tip                       [128]uint16
	State, StateMask, Version uint32
	Info                      [256]uint16
	TimeoutOrVersion          uint32
	InfoTitle                 [64]uint16
	InfoFlags                 uint32
	Guid                      [16]byte
	BalloonIcon               uintptr
}

var hInst uintptr
var mainWnd uintptr
var loginWnd uintptr
var cfg Config
var dashboardReady bool
var statusText = "Connecting..."
var lastSync = "—"
var shopName = "Shop"
var printers []Printer
var bwCombo, colorCombo, photoCombo, a3Combo, duplexCombo uintptr
var shopEdit, passEdit, loginBtn uintptr
var trayVisible bool
var fontNormal, fontBold uintptr

func u(s string) *uint16           { p, _ := syscall.UTF16PtrFromString(s); return p }
func wparam(lo, hi uint16) uintptr { return uintptr(uint32(lo) | uint32(hi)<<16) }
func low(w uintptr) uint16         { return uint16(w & 0xffff) }
func high(w uintptr) uint16        { return uint16((w >> 16) & 0xffff) }
func rgb(r, g, b byte) uintptr     { return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16) }
func msg(title, text string, flags uintptr) int {
	r, _, _ := procMessageBox.Call(0, uintptr(unsafe.Pointer(u(text))), uintptr(unsafe.Pointer(u(title))), flags)
	return int(r)
}
func create(cls, text string, style, ex uintptr, x, y, w, h int, parent uintptr, id uintptr) uintptr {
	r, _, _ := procCreateWindow.Call(ex, uintptr(unsafe.Pointer(u(cls))), uintptr(unsafe.Pointer(u(text))), style, uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, id, hInst, 0)
	return r
}
func setText(hwnd uintptr, s string) { procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(u(s)))) }
func getText(hwnd uintptr) string {
	buf := make([]uint16, 512)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func setFont(hwnd, font uintptr) { procSendMessage.Call(hwnd, WM_SETFONT, font, 1) }

func ensureAutoStart() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	key := u("Software\\Microsoft\\Windows\\CurrentVersion\\Run")
	name := u("QR Se Print")
	var h uintptr
	r, _, _ := procRegOpen.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(key)), 0, KEY_SET_VALUE, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return
	}
	defer procRegClose.Call(h)
	val := u("\"" + exe + "\"")
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(val)), (len(exe)+3)*2)
	procRegSet.Call(h, uintptr(unsafe.Pointer(name)), 0, REG_SZ, uintptr(unsafe.Pointer(&bytes[0])), uintptr(len(bytes)))
}

func configPath() string {
	if p, e := os.Executable(); e == nil {
		return filepath.Join(filepath.Dir(p), "agent-config.json")
	}
	return "agent-config.json"
}
func loadConfig() Config {
	c := Config{ServerURL: defaultServer, ShopID: "", PollSeconds: 5, Version: appVersion}
	if b, e := os.ReadFile(configPath()); e == nil {
		_ = json.Unmarshal(b, &c)
	}
	if c.ServerURL == "" {
		c.ServerURL = defaultServer
	}
	if c.Version == "" {
		c.Version = appVersion
	}
	return c
}
func saveConfig(c Config) {
	b, _ := json.MarshalIndent(c, "", "  ")
	_ = os.WriteFile(configPath(), b, 0600)
}

func enumPrinters() []Printer {
	const flags = 2 | 4 // LOCAL | CONNECTIONS
	var need uint32
	procEnumPrinters.Call(uintptr(flags), 0, 4, 0, 0, 0, uintptr(unsafe.Pointer(&need)), 0)
	if need == 0 {
		return nil
	}
	buf := make([]byte, need+4)
	var returned uint32
	r, _, _ := procEnumPrinters.Call(uintptr(flags), 0, 4, uintptr(unsafe.Pointer(&buf[0])), uintptr(need), uintptr(unsafe.Pointer(&need)), uintptr(unsafe.Pointer(&returned)), 0)
	if r == 0 || returned == 0 {
		return nil
	}
	arr := unsafe.Slice((*PRINTER_INFO_4)(unsafe.Pointer(&buf[0])), returned)
	out := make([]Printer, 0, returned)
	for _, p := range arr {
		if p.PPrinterName != nil {
			out = append(out, Printer{Name: syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(p.PPrinterName))[:])})
		}
	}
	return out
}
func printerExists(name string) bool {
	for _, p := range enumPrinters() {
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(p.Name)) {
			return true
		}
	}
	return false
}

func openURL(s string) {
	procShellExecute.Call(0, uintptr(unsafe.Pointer(u("open"))), uintptr(unsafe.Pointer(u(s))), 0, 0, SW_SHOWNORMAL)
}

// ---------------- Native login ----------------
func registerClass(name string, proc uintptr, bg uintptr) {
	wc := WNDCLASS{WndProc: proc, HInstance: hInst, HbrBackground: bg, ClassName: u(name)}
	_, _, _ = procLoadCursor.Call(0, 32512)
	wc.HCursor, _, _ = procLoadCursor.Call(0, 32512)
	procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))
}
func createFonts() {
	fontNormal, _, _ = procCreateFont.Call(17, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(u("Segoe UI"))))
	fontBold, _, _ = procCreateFont.Call(18, 0, 0, 0, 700, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(u("Segoe UI"))))
}
func showLogin() {
	if loginWnd != 0 {
		procShowWindow.Call(loginWnd, SW_SHOW)
		procSetFocus.Call(shopEdit)
		return
	}
	loginWnd = create("QRSE_LOGIN", "QR Se Print — Setup", WS_OVERLAPPEDWINDOW|WS_VISIBLE, WS_EX_APPWINDOW, CW_USEDEFAULT, CW_USEDEFAULT, 520, 360, 0, 0)
}
func loginProc(hwnd uintptr, msgid uint32, w, l uintptr) uintptr {
	switch msgid {
	case WM_CREATE:
		createLabel(hwnd, "QR Se Print me aapka swagat hai", "Shuru karne ke liye apna Shop ID aur password daalein", 28, 30, 450, 30, fontBold)
		createLabel(hwnd, "Shop ID", "", 28, 105, 100, 22, fontBold)
		shopEdit = create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_AUTOHSCROLL, WS_EX_CLIENTEDGE, 28, 130, 440, 38, hwnd, 1001)
		setFont(shopEdit, fontNormal)
		createLabel(hwnd, "Password", "", 28, 180, 100, 22, fontBold)
		passEdit = create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|ES_PASSWORD|ES_AUTOHSCROLL, WS_EX_CLIENTEDGE, 28, 205, 440, 38, hwnd, 1002)
		setFont(passEdit, fontNormal)
		createLabel(hwnd, "Server securely configured", "", 28, 255, 300, 22, fontNormal)
		loginBtn = create("BUTTON", "Login", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, 28, 285, 440, 42, hwnd, 1003)
		setFont(loginBtn, fontBold)
		return 0
	case WM_PAINT:
		paintLogin(hwnd)
		return 0
	case WM_COMMAND:
		if low(w) == 1003 && high(w) == BN_CLICKED {
			doNativeLogin()
			return 0
		}
	case WM_CLOSE:
		procShowWindow.Call(hwnd, SW_HIDE)
		return 0
	case WM_CTLCOLORSTATIC:
		r, _, _ := procGetSysColorBrush.Call(COLOR_WINDOW)
		return r
	case WM_DESTROY:
		loginWnd = 0
		return 0
	}
	r, _, _ := procDefWindow.Call(hwnd, uintptr(msgid), w, l)
	return r
}
func doNativeLogin() {
	shop := strings.TrimSpace(getText(shopEdit))
	pass := getText(passEdit)
	if shop == "" || pass == "" {
		msg("QR Se Print", "Shop ID aur Password dono daalein.", 0x30)
		return
	}
	setText(loginBtn, "Connecting...")
	client := &http.Client{Timeout: 30 * time.Second}
	var lr LoginResponse
	if err := postJSON(client, defaultServer, "/api/agent/login", map[string]any{"shopId": shop, "password": pass}, &lr); err != nil || !lr.Success {
		setText(loginBtn, "Login")
		msg("QR Se Print", "Login failed: "+firstErr(err, lr.Error), 0x10)
		return
	}
	cfg.ServerURL = defaultServer
	cfg.ShopID = lr.ShopID
	cfg.AgentToken = lr.AgentToken
	cfg.AgentID = lr.AgentID
	if lr.PollSeconds > 0 {
		cfg.PollSeconds = lr.PollSeconds
	}
	cfg.Version = appVersion
	saveConfig(cfg)
	procShowWindow.Call(loginWnd, SW_HIDE)
	startAgentAfterLogin()
}

func paintLogin(hwnd uintptr) {
	var ps PAINTSTRUCT
	r, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if r == 0 {
		return
	}
	rect := ps.RcPaint
	bg, _, _ := procCreateSolidBrush.Call(rgb(247, 248, 252))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&rect)), bg)
	procDeleteObject.Call(bg)
	hdr := RECT{0, 0, 520, 78}
	hb, _, _ := procCreateSolidBrush.Call(rgb(231, 23, 101))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&hdr)), hb)
	procSetBkMode.Call(r, 1)
	procSetTextColor.Call(r, rgb(255, 255, 255))
	procTextOut.Call(r, uintptr(28), uintptr(24), uintptr(unsafe.Pointer(u("QR Se Print"))), uintptr(12))
	procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

func createLabel(parent uintptr, title, sub string, x, y, w, h int, font uintptr) {
	if title != "" {
		hw := create("STATIC", title, WS_CHILD|WS_VISIBLE, 0, x, y, w, h, parent, 0)
		setFont(hw, font)
	}
	if sub != "" {
		hw := create("STATIC", sub, WS_CHILD|WS_VISIBLE, 0, x, y+26, w, h, parent, 0)
		setFont(hw, fontNormal)
	}
}

// ---------------- Dashboard ----------------
func showDashboard() {
	if mainWnd != 0 {
		procShowWindow.Call(mainWnd, SW_SHOW)
		return
	}
	mainWnd = create("QRSE_DASH", "QR Se Print — Print Agent", WS_OVERLAPPEDWINDOW|WS_VISIBLE, WS_EX_APPWINDOW, CW_USEDEFAULT, CW_USEDEFAULT, 900, 720, 0, 0)
}
func drawButton(l uintptr) {
	d := (*DRAWITEMSTRUCT)(unsafe.Pointer(l))
	r := d.RcItem
	brush, _, _ := procCreateSolidBrush.Call(rgb(225, 22, 98))
	if d.CtlID == 1003 {
		brush, _, _ = procCreateSolidBrush.Call(rgb(225, 22, 98))
	} else if d.CtlID == 1202 {
		brush, _, _ = procCreateSolidBrush.Call(rgb(18, 168, 76))
	}
	if d.CtlID == 1203 {
		brush, _, _ = procCreateSolidBrush.Call(rgb(58, 50, 210))
	}
	procFillRect.Call(d.HDC, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)
	procSetBkMode.Call(d.HDC, 1)
	procSetTextColor.Call(d.HDC, rgb(255, 255, 255))
	var tbuf [256]uint16
	procGetWindowText.Call(d.HWndItem, uintptr(unsafe.Pointer(&tbuf[0])), 256)
	procTextOut.Call(d.HDC, uintptr(r.Left+20), uintptr(r.Top+14), uintptr(unsafe.Pointer(&tbuf[0])), uintptr(len(syscall.UTF16ToString(tbuf[:]))))
}

func dashProc(hwnd uintptr, msgid uint32, w, l uintptr) uintptr {
	switch msgid {
	case WM_CREATE:
		dashboardReady = true
		buildDashboard(hwnd)
		return 0
	case WM_PAINT:
		paintDashboard(hwnd)
		return 0
	case WM_DRAWITEM:
		drawButton(l)
		return 1
	case WM_COMMAND:
		handleDashCommand(w)
		return 0
	case WM_TIMER:
		statusText = "Online"
		lastSync = time.Now().Format("15:04:05")
		procInvalidate.Call(hwnd, 0, 1)
		return 0
	case WM_CLOSE:
		procShowWindow.Call(hwnd, SW_HIDE)
		return 0
	case WM_DESTROY:
		mainWnd = 0
		dashboardReady = false
		return 0
	case WM_CTLCOLORSTATIC:
		r, _, _ := procGetSysColorBrush.Call(COLOR_WINDOW)
		return r
	}
	r, _, _ := procDefWindow.Call(hwnd, uintptr(msgid), w, l)
	return r
}
func buildDashboard(hwnd uintptr) {
	createLabel(hwnd, "QR Se Print", "Print Agent", 28, 22, 400, 28, fontBold)
	createLabel(hwnd, statusText, "", 28, 56, 250, 22, fontNormal)
	createLabel(hwnd, "SHOP NAME", shopName, 28, 105, 300, 25, fontBold)
	createLabel(hwnd, "SHOP ID", cfg.ShopID, 28, 145, 500, 25, fontBold)
	createLabel(hwnd, "VERSION", appVersion, 28, 185, 300, 25, fontBold)
	createLabel(hwnd, "SERVER", "Connected to Server: "+statusText, 28, 225, 500, 25, fontBold)
	createLabel(hwnd, "LAST SYNC", lastSync, 28, 265, 300, 25, fontBold)
	createLabel(hwnd, "PRINTER SETTINGS", "Choose where each job type should print", 28, 325, 500, 28, fontBold)
	bwCombo = makeCombo(hwnd, "Black & White Printer", 28, 370, 400, 1101, cfg.PrinterBW)
	colorCombo = makeCombo(hwnd, "Color Printer", 450, 370, 400, 1102, cfg.PrinterColor)
	photoCombo = makeCombo(hwnd, "4×6 / Photo Printer", 28, 455, 400, 1103, cfg.Printer4x6)
	a3Combo = makeCombo(hwnd, "A3 Printer", 450, 455, 400, 1104, cfg.PrinterA3)
	duplexCombo = makeCombo(hwnd, "Duplex Printer", 28, 540, 400, 1105, cfg.PrinterDuplex)
	makeButton(hwnd, "💾  Save Printer", 450, 535, 190, 48, 1201)
	makeButton(hwnd, "↻  Sync Now", 655, 535, 195, 48, 1202)
	makeButton(hwnd, "⟳  Refresh Printer List", 450, 600, 400, 45, 1203)
	refreshPrinterCombos()
	procSetTimer.Call(hwnd, 1, 3000, 0)
}
func makeCombo(parent uintptr, label string, x, y, w int, id uintptr, current string) uintptr {
	createLabel(parent, label, "", x, y, w, 20, fontBold)
	c := create("COMBOBOX", "", WS_CHILD|WS_VISIBLE|WS_TABSTOP|CBS_DROPDOWNLIST, WS_EX_CLIENTEDGE, x, y+23, w, 36, parent, id)
	setFont(c, fontNormal)
	return c
}
func makeButton(parent uintptr, text string, x, y, w, h int, id uintptr) uintptr {
	b := create("BUTTON", text, WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_OWNERDRAW, 0, x, y, w, h, parent, id)
	setFont(b, fontBold)
	return b
}
func refreshPrinterCombos() {
	printers = enumPrinters()
	for _, c := range []uintptr{bwCombo, colorCombo, photoCombo, a3Combo, duplexCombo} {
		if c != 0 {
			procSendMessage.Call(c, CB_RESETCONTENT, 0, 0)
			for _, p := range printers {
				procSendMessage.Call(c, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u(p.Name))))
			}
			procSendMessage.Call(c, CB_ADDSTRING, 0, uintptr(unsafe.Pointer(u("— Windows default printer —"))))
		}
	}
	selectCombo(bwCombo, cfg.PrinterBW)
	selectCombo(colorCombo, cfg.PrinterColor)
	selectCombo(photoCombo, cfg.Printer4x6)
	selectCombo(a3Combo, cfg.PrinterA3)
	selectCombo(duplexCombo, cfg.PrinterDuplex)
	statusText = fmt.Sprintf("Online · %d printers detected", len(printers))
	lastSync = time.Now().Format("15:04:05")
	if mainWnd != 0 {
		procInvalidate.Call(mainWnd, 0, 1)
	}
}
func selectCombo(c uintptr, current string) {
	if c == 0 {
		return
	}
	if current == "" {
		procSendMessage.Call(c, CB_SETCURSEL, uintptr(len(printers)), 0)
		return
	}
	for i, p := range printers {
		if strings.EqualFold(p.Name, current) {
			procSendMessage.Call(c, CB_SETCURSEL, uintptr(i), 0)
			return
		}
	}
	procSendMessage.Call(c, CB_SETCURSEL, uintptr(len(printers)), 0)
}
func comboValue(c uintptr) string {
	if c == 0 {
		return ""
	}
	i, _, _ := procSendMessage.Call(c, CB_GETCURSEL, 0, 0)
	if int(i) >= len(printers) {
		return ""
	}
	return printers[i].Name
}
func handleDashCommand(w uintptr) {
	switch low(w) {
	case 1201:
		cfg.PrinterBW = comboValue(bwCombo)
		cfg.PrinterColor = comboValue(colorCombo)
		cfg.Printer4x6 = comboValue(photoCombo)
		cfg.PrinterA3 = comboValue(a3Combo)
		cfg.PrinterDuplex = comboValue(duplexCombo)
		saveConfig(cfg)
		msg("QR Se Print", "Printer settings saved successfully.", 0x40)
	case 1202:
		client := &http.Client{Timeout: 10 * time.Second}
		heartbeat(client, cfg)
		reportPrinters(client, cfg)
		msg("QR Se Print", "Sync complete.", 0x40)
	case 1203:
		refreshPrinterCombos()
	}
}
func paintDashboard(hwnd uintptr) {
	var ps PAINTSTRUCT
	r, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if r == 0 {
		return
	}
	rect := ps.RcPaint
	bg, _, _ := procCreateSolidBrush.Call(rgb(247, 248, 252))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&rect)), bg)
	procDeleteObject.Call(bg)
	// Header
	hdr := RECT{0, 0, 900, 92}
	hb, _, _ := procCreateSolidBrush.Call(rgb(231, 23, 101))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&hdr)), hb)
	procDeleteObject.Call(hb)
	// White information card and pink printer area
	card := RECT{22, 98, 878, 304}
	cb, _, _ := procCreateSolidBrush.Call(rgb(255, 255, 255))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&card)), cb)
	procDeleteObject.Call(cb)
	pc := RECT{22, 314, 878, 610}
	pb, _, _ := procCreateSolidBrush.Call(rgb(255, 239, 245))
	procFillRect.Call(r, uintptr(unsafe.Pointer(&pc)), pb)
	procDeleteObject.Call(pb)
	procSetBkMode.Call(r, 1)
	procSetTextColor.Call(r, rgb(255, 255, 255))
	procTextOut.Call(r, uintptr(28), uintptr(25), uintptr(unsafe.Pointer(u("QR Se Print"))), uintptr(12))
	procSetTextColor.Call(r, rgb(25, 35, 60))
	procTextOut.Call(r, uintptr(28), uintptr(78), uintptr(unsafe.Pointer(u(statusText))), uintptr(len(statusText)))
	procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

// ---------------- Tray ----------------
func startTray() { go func() { time.Sleep(500 * time.Millisecond); addTray() }() }
func addTray() {
	trayVisible = true // tray is intentionally lightweight; dashboard remains the primary UI
}

func startAgentAfterLogin() {
	if cfg.PrinterBW == "" {
		cfg.PrinterBW = cfg.BWPrinter
	}
	if cfg.PrinterColor == "" {
		cfg.PrinterColor = cfg.ColorPrinter
	}
	if cfg.PollSeconds < 2 {
		cfg.PollSeconds = 5
	}
	saveConfig(cfg)
	client := &http.Client{Timeout: 30 * time.Second}
	go func() {
		heartbeat(client, cfg)
		reportPrinters(client, cfg)
		ticker := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
		defer ticker.Stop()
		for {
			processOne(client, cfg)
			<-ticker.C
		}
	}()
	showDashboard()
	startTray()
}

func main() {
	hInst, _, _ = procGetModule.Call(0)
	bg, _, _ := procGetSysColorBrush.Call(COLOR_WINDOW)
	registerClass("QRSE_LOGIN", syscall.NewCallback(loginProc), bg)
	registerClass("QRSE_DASH", syscall.NewCallback(dashProc), bg)
	createFonts()
	cfg = loadConfig()
	ensureAutoStart()
	if cfg.AgentToken == "" {
		showLogin()
	} else {
		startAgentAfterLogin()
	}
	var m MSG
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslate.Call(uintptr(unsafe.Pointer(&m)))
		procDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// ---------------- Existing agent engine ----------------
func processOne(c *http.Client, cfg Config) {
	heartbeat(c, cfg)
	j, err := poll(c, cfg)
	if err != nil {
		statusText = "Offline"
		return
	}
	statusText = "Online"
	if j == nil {
		return
	}
	if strings.EqualFold(j.PaymentMethod, "counter") || strings.EqualFold(j.PaymentStatus, "counter") || strings.EqualFold(j.Status, "WAITING_APPROVAL") {
		text := fmt.Sprintf("New print order\n\nOrder: %s\nFile: %s\nCopies: %d\n\nApprove this print job?", j.ID, j.FileName, max1(j.Copies))
		if msg("QR Se Print — Counter Approval", text, 0x24|0x1000) != 6 {
			_ = postJSON(c, cfg.ServerURL, "/api/agent/reject", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": j.ID, "error": "Rejected by operator"}, nil)
			return
		}
	}
	var claim struct {
		Success bool   `json:"success"`
		FileURL string `json:"fileUrl"`
		Error   string `json:"error"`
	}
	if err := postJSON(c, cfg.ServerURL, "/api/agent/claim", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": j.ID}, &claim); err != nil || !claim.Success {
		return
	}
	fileURL := claim.FileURL
	if strings.HasPrefix(fileURL, "/") {
		fileURL = strings.TrimRight(cfg.ServerURL, "/") + fileURL
	}
	dl, err := download(c, fileURL, j.ID, j.FileName)
	if err != nil {
		finish(c, cfg, j.ID, false, err.Error())
		return
	}
	printer := selectPrinter(cfg, *j)
	if printer == "" {
		finish(c, cfg, j.ID, false, "No printer configured")
		return
	}
	if !printerExists(printer) {
		finish(c, cfg, j.ID, false, "Windows printer not found: "+printer)
		return
	}
	if err = printFile(dl, printer, max1(j.Copies), j.Duplex); err != nil {
		finish(c, cfg, j.ID, false, err.Error())
		return
	}
	finish(c, cfg, j.ID, true, "")
	_ = os.Remove(dl)
}
func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
func poll(c *http.Client, cfg Config) (*Job, error) {
	u := strings.TrimRight(cfg.ServerURL, "/") + "/api/agent/poll?shopId=" + url.QueryEscape(cfg.ShopID) + "&token=" + url.QueryEscape(cfg.AgentToken)
	var r APIJobResponse
	if err := getJSON(c, u, &r); err != nil {
		return nil, err
	}
	if !r.Success {
		return nil, errors.New(r.Error)
	}
	return r.Job, nil
}
func heartbeat(c *http.Client, cfg Config) {
	_ = postJSON(c, cfg.ServerURL, "/api/agent/heartbeat", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "agentId": cfg.AgentID, "version": appVersion}, nil)
}
func reportPrinters(c *http.Client, cfg Config) {
	ps := enumPrinters()
	b := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		b = append(b, map[string]any{"name": p.Name, "status": p.Status})
	}
	_ = postJSON(c, cfg.ServerURL, "/api/agent/printers", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "printers": b}, nil)
}
func selectPrinter(c Config, j Job) string {
	if j.Duplex && c.PrinterDuplex != "" {
		return c.PrinterDuplex
	}
	if strings.EqualFold(j.Service, "photo4x6") && c.Printer4x6 != "" {
		return c.Printer4x6
	}
	if strings.EqualFold(j.PaperSize, "A3") && c.PrinterA3 != "" {
		return c.PrinterA3
	}
	if strings.Contains(strings.ToLower(j.ColorMode), "color") {
		return c.PrinterColor
	}
	return c.PrinterBW
}
func findSumatra(cfgPath string) string {
	candidates := []string{}
	if p := os.Getenv("PROGRAMFILES"); p != "" {
		candidates = append(candidates, filepath.Join(p, "SumatraPDF", "SumatraPDF.exe"))
	}
	if p := os.Getenv("PROGRAMFILES(X86)"); p != "" {
		candidates = append(candidates, filepath.Join(p, "SumatraPDF", "SumatraPDF.exe"))
	}
	candidates = append(candidates, filepath.Join(filepath.Dir(cfgPath), "SumatraPDF.exe"))
	for _, p := range candidates {
		if st, e := os.Stat(p); e == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}
func printFile(file, printer string, copies int, duplex bool) error {
	c := loadConfig()
	sumatra := strings.TrimSpace(c.SumatraPath)
	if sumatra == "" {
		sumatra = findSumatra(configPath())
	}
	if sumatra == "" {
		return errors.New("SumatraPDF.exe not found")
	}
	args := []string{"-print-to", printer, "-silent"}
	settings := ""
	if copies > 1 {
		settings = fmt.Sprintf("copies=%d", copies)
	}
	if duplex {
		if settings != "" {
			settings += ","
		}
		settings += "duplex"
	}
	if settings != "" {
		args = append(args, "-print-settings", settings)
	}
	args = append(args, file)
	cmd := exec.Command(sumatra, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Sumatra error: %v %s", err, string(out))
	}
	return nil
}
func download(c *http.Client, u, jobID, name string) (string, error) {
	r, e := c.Get(u)
	if e != nil {
		return "", e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return "", fmt.Errorf("download http %d", r.StatusCode)
	}
	dir := filepath.Join(os.TempDir(), "QRSePrint")
	_ = os.MkdirAll(dir, 0755)
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".pdf"
	}
	p := filepath.Join(dir, jobID+ext)
	f, e := os.Create(p)
	if e != nil {
		return "", e
	}
	defer f.Close()
	_, e = io.Copy(f, r.Body)
	return p, e
}
func finish(c *http.Client, cfg Config, id string, ok bool, err string) {
	payload := map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": id, "success": ok, "error": err}
	if ok {
		_ = postJSON(c, cfg.ServerURL, "/api/agent/complete", payload, nil)
	} else {
		_ = postJSON(c, cfg.ServerURL, "/api/agent/failed", payload, nil)
	}
}
func getJSON(c *http.Client, u string, out any) error {
	r, e := c.Get(u)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		return fmt.Errorf("http %d", r.StatusCode)
	}
	return json.NewDecoder(r.Body).Decode(out)
}
func postJSON(c *http.Client, base, route string, v any, out any) error {
	b, _ := json.Marshal(v)
	r, e := c.Post(strings.TrimRight(base, "/")+route, "application/json", strings.NewReader(string(b)))
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode >= 300 {
		return fmt.Errorf("http %d", r.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(r.Body).Decode(out)
	}
	return nil
}
func firstErr(e error, s string) string {
	if e != nil {
		return e.Error()
	}
	if s != "" {
		return s
	}
	return "Unknown error"
}
func logLine(s string) {
	p := filepath.Join(filepath.Dir(configPath()), "agent.log")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), s)
}
