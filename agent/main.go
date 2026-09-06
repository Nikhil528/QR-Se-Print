//go:build windows

package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

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
	Success    bool   `json:"success"`
	ShopID     string `json:"shopId"`
	AgentToken string `json:"agentToken"`
	AgentID    string `json:"agentId"`
	Error      string `json:"error"`
}
type ClaimResponse struct {
	Success bool   `json:"success"`
	FileURL string `json:"fileUrl"`
	Error   string `json:"error"`
}

type Printer struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

var user32 = syscall.NewLazyDLL("user32.dll")

var stateMu sync.RWMutex
var runtimeState = struct {
	connected       bool
	lastSync        string
	jobs            []map[string]any
	counterApproval bool
}{counterApproval: true}
var messageBox = user32.NewProc("MessageBoxW")

func msg(title, text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	r, _, _ := messageBox.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return int(r)
}

func main() {
	cfg := loadConfig()
	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "--logout") {
		cfg.AgentToken = ""
		cfg.AgentID = ""
		saveConfig(cfg)
		msg("QR Se Print", "Logged out. Start the agent again to login.", 0x40)
		return
	}
	if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "--login") {
		cfg = doLogin(cfg)
		if cfg.AgentToken == "" {
			return
		}
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "https://bvv-djql.onrender.com"
	}
	if cfg.PrinterBW == "" {
		cfg.PrinterBW = cfg.BWPrinter
	}
	if cfg.PrinterColor == "" {
		cfg.PrinterColor = cfg.ColorPrinter
	}
	if cfg.PollSeconds < 2 {
		cfg.PollSeconds = 5
	}
	if cfg.Version == "" {
		cfg.Version = "9.1.0"
	}
	if cfg.AgentToken == "" {
		cfg = doLogin(cfg)
		if cfg.AgentToken == "" {
			return
		}
	}
	showDemoUpgradePrompt(cfg)
	logLine("========================================")
	logLine("QR Se Print Agent starting v" + cfg.Version)
	logLine("Server: " + cfg.ServerURL)
	logLine("Shop: " + cfg.ShopID)
	logLine("B&W Printer: " + cfg.PrinterBW)
	logLine("Color Printer: " + cfg.PrinterColor)
	logLine("4x6 Printer: " + cfg.Printer4x6)
	logLine("A3 Printer: " + cfg.PrinterA3)
	logLine("Duplex Printer: " + cfg.PrinterDuplex)
	logLine("PollSeconds: " + fmt.Sprint(cfg.PollSeconds))
	logLine("========================================")
	client := &http.Client{Timeout: 30 * time.Second}
	go startNativeDashboard(&cfg, client)
	startControlServer(&cfg, client)
	ensureAutoStart()
	launchTray()
	heartbeat(client, cfg)
	reportPrinters(client, cfg)
	ticker := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
	defer ticker.Stop()
	for {
		processOne(client, cfg)
		<-ticker.C
	}
}

func configPath() string {
	if p, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(p), "agent-config.json")
	}
	return "agent-config.json"
}
func loadConfig() Config {
	c := Config{ServerURL: "https://bvv-djql.onrender.com", ShopID: "DEMO", PollSeconds: 5, Version: "9.0.0"}
	b, e := os.ReadFile(configPath())
	if e == nil {
		_ = json.Unmarshal(b, &c)
	}
	return c
}
func saveConfig(c Config) {
	b, _ := json.MarshalIndent(c, "", "  ")
	_ = os.WriteFile(configPath(), b, 0600)
}

func doLogin(cfg Config) Config {
	// Native Windows login dialog; no PowerShell dependency.
	result := nativeLoginDialog(cfg.ServerURL, cfg.ShopID)
	if result.cancelled || result.shop == "" || result.password == "" {
		return cfg
	}
	server := strings.TrimRight(strings.TrimSpace(result.server), "/")
	if server == "" {
		server = cfg.ServerURL
	}
	client := &http.Client{Timeout: 30 * time.Second}
	var lr LoginResponse
	if err := postJSON(client, server, "/api/agent/login", map[string]any{"shopId": result.shop, "password": result.password}, &lr); err != nil || !lr.Success {
		msg("QR Se Print", "Login failed: "+firstErr(err, lr.Error), 0x10)
		return cfg
	}
	cfg.ServerURL = server
	cfg.ShopID = lr.ShopID
	cfg.AgentToken = lr.AgentToken
	cfg.AgentID = lr.AgentID
	cfg.Version = "9.1.0"
	saveConfig(cfg)
	msg("QR Se Print", "Login successful\n\nShop: "+cfg.ShopID+"\nAgent is ready for print jobs.", 0x40)
	return cfg
}

type loginResult struct {
	server, shop, password string
	cancelled              bool
}

func showDemoUpgradePrompt(cfg Config) {
	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(cfg.ShopID)), "DEMO_") {
		return
	}
	text := "Aapki shop abhi DEMO Shop ID par chal rahi hai.\nDemo khatam hone hi printing ruk jayegi.\n\nYes  =  Plan dekho (browser me khulega)\nNo   =  Abhi nahi"
	r := msg("QR Se Print - Demo", text, 0x24|0x1000)
	if r == 6 {
		_ = exec.Command("cmd", "/c", "start", "", "https://bvv-djql.onrender.com/").Start()
	}
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

func processOne(c *http.Client, cfg Config) {
	heartbeat(c, cfg)
	j, err := poll(c, cfg)
	if err != nil {
		logLine("POLL ERROR: " + err.Error())
		return
	}
	if j == nil {
		return
	}
	logLine(fmt.Sprintf("JOB RECEIVED: %s file=%s copies=%d color=%s status=%s", j.ID, j.FileName, j.Copies, j.ColorMode, j.Status))
	stateMu.Lock()
	runtimeState.jobs = append([]map[string]any{{"id": j.ID, "file": j.FileName, "status": "Received", "ok": false}}, runtimeState.jobs...)
	if len(runtimeState.jobs) > 20 {
		runtimeState.jobs = runtimeState.jobs[:20]
	}
	stateMu.Unlock()
	if runtimeCounterApproval() && (strings.EqualFold(j.PaymentMethod, "counter") || strings.EqualFold(j.PaymentStatus, "counter") || strings.EqualFold(j.Status, "WAITING_APPROVAL")) {
		text := fmt.Sprintf("New print order\n\nOrder: %s\nFile: %s\nCopies: %d\n\nApprove this print job?", j.ID, j.FileName, max1(j.Copies))
		if msg("QR Se Print — Counter Approval", text, 0x24|0x1000) != 6 {
			_ = postJSON(c, cfg.ServerURL, "/api/agent/reject", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": j.ID, "error": "Rejected by operator"}, nil)
			logLine("COUNTER JOB REJECTED: " + j.ID)
			return
		}
	}
	var claim ClaimResponse
	if err := postJSON(c, cfg.ServerURL, "/api/agent/claim", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": j.ID}, &claim); err != nil || !claim.Success {
		logLine("CLAIM FAILED: " + firstErr(err, claim.Error))
		return
	}
	fileURL := claim.FileURL
	if strings.HasPrefix(fileURL, "/") {
		fileURL = strings.TrimRight(cfg.ServerURL, "/") + fileURL
	}
	dl, err := download(c, fileURL, j.ID, j.FileName)
	if err != nil {
		finish(c, cfg, j.ID, false, err.Error())
		msg("QR Se Print", "Download failed: "+err.Error(), 0x10)
		return
	}
	_ = postJSON(c, cfg.ServerURL, "/api/jobs/"+url.PathEscape(j.ID)+"/downloaded", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "jobId": j.ID}, nil)
	logLine("DOWNLOADED: " + dl)
	printer := selectPrinter(cfg, *j)
	if printer == "" {
		err = errors.New("No printer configured for this job type")
		finish(c, cfg, j.ID, false, err.Error())
		return
	}
	if !printerExists(printer) {
		err = fmt.Errorf("Windows printer not found: %s", printer)
		finish(c, cfg, j.ID, false, err.Error())
		msg("QR Se Print", "Printer not found:\n"+printer, 0x10)
		return
	}
	logLine("SELECTED PRINTER: " + printer)
	if err = printFile(dl, printer, max1(j.Copies), j.Duplex); err != nil {
		logLine("PRINT FAILED: " + err.Error())
		finish(c, cfg, j.ID, false, err.Error())
		msg("QR Se Print", "Print failed:\n"+err.Error(), 0x10)
		return
	}
	logLine("PRINT VERIFIED/SUBMITTED: " + j.ID)
	finish(c, cfg, j.ID, true, "")
	logLine("COMPLETE: " + j.ID + " (server will delete file)")
	stateMu.Lock()
	for i := range runtimeState.jobs {
		if runtimeState.jobs[i]["id"] == j.ID {
			runtimeState.jobs[i]["status"] = "Printed successfully"
			runtimeState.jobs[i]["ok"] = true
		}
	}
	stateMu.Unlock()
	msg("QR Se Print", "Order "+j.ID+" printed successfully.", 0x40)
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
	err := postJSON(c, cfg.ServerURL, "/api/agent/heartbeat", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "agentId": cfg.AgentID, "version": cfg.Version}, nil)
	stateMu.Lock()
	runtimeState.connected = (err == nil)
	runtimeState.lastSync = time.Now().Format("15:04:05")
	stateMu.Unlock()
}
func reportPrinters(c *http.Client, cfg Config) {
	ps := enumeratePrinters()
	b := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		b = append(b, map[string]any{"name": p.Name, "status": p.Status})
	}
	_ = postJSON(c, cfg.ServerURL, "/api/agent/printers", map[string]any{"shopId": cfg.ShopID, "token": cfg.AgentToken, "printers": b}, nil)
	for _, p := range ps {
		logLine("PRINTER: " + p.Name + " STATUS=" + p.Status)
	}
}
func enumeratePrinters() []Printer {
	// Native Winspool enumeration. This avoids spawning PowerShell just to read printers.
	const PRINTER_ENUM_LOCAL = 2
	const PRINTER_ENUM_CONNECTIONS = 4
	var needed, returned uint32
	enum := syscall.NewLazyDLL("winspool.drv").NewProc("EnumPrintersW")
	enum.Call(PRINTER_ENUM_LOCAL|PRINTER_ENUM_CONNECTIONS, 0, 4, 0, 0, uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned)))
	if needed == 0 {
		return nil
	}
	buf := make([]byte, needed)
	r, _, _ := enum.Call(PRINTER_ENUM_LOCAL|PRINTER_ENUM_CONNECTIONS, 0, 4, uintptr(unsafe.Pointer(&buf[0])), uintptr(needed), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&returned)))
	if r == 0 || returned == 0 {
		return nil
	}
	type PRINTER_INFO_4W struct {
		PPrinterName *uint16
		PServerName  *uint16
		Attributes   uint32
	}
	size := unsafe.Sizeof(PRINTER_INFO_4W{})
	out := make([]Printer, 0, returned)
	for i := uint32(0); i < returned; i++ {
		pi := (*PRINTER_INFO_4W)(unsafe.Pointer(&buf[uintptr(i)*size]))
		if pi.PPrinterName != nil {
			out = append(out, Printer{Name: syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(pi.PPrinterName))[:]), Status: "Ready"})
		}
	}
	return out
}

func printerExists(name string) bool {
	target := strings.TrimSpace(name)
	if target == "" {
		return false
	}
	for _, p := range enumeratePrinters() {
		if strings.EqualFold(strings.TrimSpace(p.Name), target) {
			return true
		}
	}
	return false
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
func printerJobCount(name string) int {
	// Queue polling is diagnostic only. Do not spawn PowerShell and never use a
	// failed queue query as a reason to send a second print.
	return -1
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
	cfg := loadConfig()
	sumatra := strings.TrimSpace(cfg.SumatraPath)
	if sumatra == "" || func() bool { _, e := os.Stat(sumatra); return e != nil }() {
		sumatra = findSumatra(configPath())
	}
	if sumatra == "" {
		return fmt.Errorf("SumatraPDF.exe not found; install it in C:\\Program Files\\SumatraPDF or place it beside the Agent")
	}

	// IMPORTANT: Use the exact command proven to work manually on this PC.
	// Never send a second fallback print: if queue polling is unreliable, a fallback
	// would create duplicate physical prints.
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

	before := printerJobCount(printer)
	logLine(fmt.Sprintf("SPOOLER BEFORE: printer=%s jobs=%d", printer, before))
	logLine("SUMATRA PATH: " + sumatra)
	logLine("SUMATRA: " + sumatra + " " + strings.Join(args, " "))

	cmd := exec.Command(sumatra, args...)
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		logLine("SUMATRA OUTPUT: " + strings.TrimSpace(string(out)))
	}
	if err != nil {
		return fmt.Errorf("Sumatra error: %v", err)
	}
	logLine("SUMATRA EXIT CODE: 0")

	// Queue entries can disappear very quickly on some Windows printer drivers.
	// Therefore queue visibility is diagnostic only, not a second print trigger.
	if before >= 0 {
		deadline := time.Now().Add(5 * time.Second)
		verified := false
		for time.Now().Before(deadline) {
			n := printerJobCount(printer)
			if n >= 0 && n > before {
				verified = true
				logLine(fmt.Sprintf("SPOOLER OBSERVED: printer=%s jobs=%d", printer, n))
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if !verified {
			logLine("SPOOLER NOT OBSERVED: queue item may have been too brief to observe; no duplicate fallback will be sent")
		}
	} else {
		logLine("SPOOLER CHECK UNAVAILABLE: Get-PrintJob query failed; trusting Sumatra exit code and avoiding duplicate fallback")
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
func logLine(s string) {
	p := filepath.Join(filepath.Dir(configPath()), "agent.log")
	f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), s)
}

func runtimeCounterApproval() bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return runtimeState.counterApproval
}

func startControlServer(cfg *Config, client *http.Client) {
	mux := http.NewServeMux()
	mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
		if nativeHwnd != 0 {
			procShowWindow.Call(nativeHwnd, swShow)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/reconnect", func(w http.ResponseWriter, r *http.Request) {
		heartbeat(client, *cfg)
		reportPrinters(client, *cfg)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		heartbeat(client, *cfg)
		reportPrinters(client, *cfg)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/counter", func(w http.ResponseWriter, r *http.Request) {
		var x struct {
			Enabled bool `json:"enabled"`
		}
		_ = json.NewDecoder(r.Body).Decode(&x)
		stateMu.Lock()
		runtimeState.counterApproval = x.Enabled
		stateMu.Unlock()
		logLine(fmt.Sprintf("Counter approval: %v", x.Enabled))
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		_ = exec.Command("notepad.exe", filepath.Join(filepath.Dir(configPath()), "agent.log")).Start()
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/change-shop", func(w http.ResponseWriter, r *http.Request) {
		next := doLogin(*cfg)
		if next.AgentToken != "" {
			*cfg = next
			heartbeat(client, *cfg)
			reportPrinters(client, *cfg)
			if nativeShop != 0 {
				nativeSetText(nativeShop, next.ShopID)
			}
			if nativeVersion != 0 {
				nativeSetText(nativeVersion, "v"+next.Version)
			}
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	go func() {
		if err := http.ListenAndServe("127.0.0.1:17845", mux); err != nil {
			logLine("Control server stopped: " + err.Error())
		}
	}()
}

func ensureAutoStart() {
	exe, _ := os.Executable()
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	cmd := `reg add "` + key + `" /v "QR Se Print Agent" /t REG_SZ /d """` + exe + `""" /f`
	_ = exec.Command("cmd", "/c", cmd).Run()
	logLine("Auto-start ensured via HKCU Run")
}

// Native dashboard and tray. No browser and no PowerShell are used for the agent UI.
var (
	nativeHwnd                                                        uintptr
	nativeBW, nativeColor, nativeP4, nativeA3, nativeDup              uintptr
	nativeStatus, nativeServer, nativeSync, nativeShop, nativeVersion uintptr
	nativeCfg                                                         *Config
	nativeClient                                                      *http.Client
	nativeCounterButton                                               uintptr
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	shell32                 = syscall.NewLazyDLL("shell32.dll")
	gdi32                   = syscall.NewLazyDLL("gdi32.dll")
	procGetModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	procGetMessage          = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessage     = user32.NewProc("DispatchMessageW")
	procRegisterClassEx     = user32.NewProc("RegisterClassExW")
	procCreateWindowEx      = user32.NewProc("CreateWindowExW")
	procDefWindowProc       = user32.NewProc("DefWindowProcW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procUpdateWindow        = user32.NewProc("UpdateWindow")
	procSetWindowText       = user32.NewProc("SetWindowTextW")
	procSendMessage         = user32.NewProc("SendMessageW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procPostQuitMessage     = user32.NewProc("PostQuitMessage")
	procSetFocus            = user32.NewProc("SetFocus")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procBeginPaint          = user32.NewProc("BeginPaint")
	procEndPaint            = user32.NewProc("EndPaint")
	procFillRect            = user32.NewProc("FillRect")
	procCreateSolidBrush    = gdi32.NewProc("CreateSolidBrush")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procSelectObject        = gdi32.NewProc("SelectObject")
	procCreateFont          = gdi32.NewProc("CreateFontW")
	procSetTextColor        = gdi32.NewProc("SetTextColor")
	procSetBkMode           = gdi32.NewProc("SetBkMode")
	procDrawText            = user32.NewProc("DrawTextW")
	procShellExecute        = shell32.NewProc("ShellExecuteW")
	procTrackPopupMenu      = user32.NewProc("TrackPopupMenu")
	procCreatePopupMenu     = user32.NewProc("CreatePopupMenu")
	procAppendMenu          = user32.NewProc("AppendMenuW")
	procDestroyMenu         = user32.NewProc("DestroyMenu")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessage         = user32.NewProc("PostMessageW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
)

type wPoint struct{ x, y int32 }
type wMsg struct {
	hwnd           uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	pt             wPoint
	lPrivate       uint32
}
type wWndClassEx struct {
	cbSize                                   uint32
	style                                    uint32
	wndProc                                  uintptr
	cbClsExtra, cbWndExtra                   int32
	hInstance, hIcon, hCursor, hbrBackground uintptr
	menuName, className                      *uint16
	hIconSm                                  uintptr
}
type wRect struct{ left, top, right, bottom int32 }
type wPaint struct {
	hdc         uintptr
	erase       int32
	rcPaint     wRect
	restore     int32
	update      int32
	rgbReserved [32]byte
}
type drawItem struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	hwndItem, hdc                                 uintptr
	rcItem                                        wRect
	itemData                                      uintptr
}
type trayIconData struct {
	cbSize                          uint32
	hwnd                            uintptr
	uID                             uint32
	uFlags, uCallbackMessage, hIcon uintptr
	tip                             [128]uint16
	state, stateMask                uint32
	info                            [256]uint16
	timeout                         uint32
	infoTitle                       [64]uint16
	infoFlags                       uint32
	guid                            [16]byte
	balloonIcon                     uint32
}
type loginState struct {
	hwnd, editShop, editPass, editServer, btnLogin, btnCancel uintptr
	result                                                    loginResult
}

const (
	wmDestroy        = 0x0002
	wmPaint          = 0x000F
	wmClose          = 0x0010
	wmCommand        = 0x0111
	wmDrawItem       = 0x002B
	wmCtlColorStatic = 0x0138
	wmCtlColorEdit   = 0x0133
	cbAddString      = 0x0143
	cbResetContent   = 0x014B
	cbGetCurSel      = 0x0147
	cbSetCurSel      = 0x014E
	cbGetLBText      = 0x0148
	bnClicked        = 0
	wsOverlapped     = 0x00CF0000
	wsChild          = 0x40000000
	wsVisible        = 0x10000000
	wsBorder         = 0x00800000
	wsTabstop        = 0x00010000
	ssLeft           = 0
	bsOwnerDraw      = 0x0000000B
	cbxDropdownList  = 0x0003
	swShow           = 5
	swHide           = 0
	idBW             = 101
	idColor          = 102
	idP4             = 103
	idA3             = 104
	idDup            = 105
	idSave           = 201
	idSync           = 202
	idRefresh        = 203
	idPlan           = 204
	idReconnect      = 205
	idCounter        = 206
	idChangeShop     = 207
	idLogs           = 208
	idLogin          = 301
	idCancelLogin    = 302
	idLoginServer    = 303
	idLoginShop      = 304
	idLoginPass      = 305
	trayMsg          = 0x8001
	trayUID          = 77
	trayOpen         = 4001
	traySync         = 4002
	trayReconnect    = 4003
	trayExit         = 4004
)

func utf16(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func nativeCreate(parent uintptr, class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	r, _, _ := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(utf16(class))), uintptr(unsafe.Pointer(utf16(text))), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, uintptr(id), 0, 0)
	return r
}
func nativeLabel(parent uintptr, text string, x, y, w, h int32) uintptr {
	return nativeCreate(parent, "STATIC", text, wsChild|wsVisible|ssLeft, x, y, w, h, 0)
}
func nativeButton(parent uintptr, text string, x, y, w, h int32, id int) uintptr {
	return nativeCreate(parent, "BUTTON", text, wsChild|wsVisible|wsTabstop|bsOwnerDraw, x, y, w, h, id)
}
func nativeCombo(parent uintptr, x, y, w, h int32, id int) uintptr {
	return nativeCreate(parent, "COMBOBOX", "", wsChild|wsVisible|wsTabstop|wsBorder|cbxDropdownList, x, y, w, h, id)
}
func nativeEdit(parent uintptr, text string, x, y, w, h int32, id int, password bool) uintptr {
	st := uint32(wsChild | wsVisible | wsTabstop | wsBorder)
	if password {
		st |= 0x20
	}
	return nativeCreate(parent, "EDIT", text, st, x, y, w, h, id)
}
func nativeSetText(hwnd uintptr, text string) {
	if hwnd != 0 {
		procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(utf16(text))))
	}
}
func nativeAddCombo(hwnd uintptr, text string) {
	procSendMessage.Call(hwnd, cbAddString, 0, uintptr(unsafe.Pointer(utf16(text))))
}
func nativeGetCombo(hwnd uintptr) string {
	idx, _, _ := procSendMessage.Call(hwnd, cbGetCurSel, 0, 0)
	if int32(idx) == -1 {
		return ""
	}
	buf := make([]uint16, 512)
	procSendMessage.Call(hwnd, cbGetLBText, idx, uintptr(unsafe.Pointer(&buf[0])))
	return syscall.UTF16ToString(buf)
}
func nativeSetCombo(hwnd uintptr, items []string, selected string) {
	procSendMessage.Call(hwnd, cbResetContent, 0, 0)
	nativeAddCombo(hwnd, "— Windows ka default printer —")
	sel := 0
	for i, p := range items {
		nativeAddCombo(hwnd, p)
		if strings.EqualFold(p, selected) {
			sel = i + 1
		}
	}
	procSendMessage.Call(hwnd, cbSetCurSel, uintptr(sel), 0)
}
func nativePrinterNames() []string {
	ps := enumeratePrinters()
	a := make([]string, 0, len(ps))
	for _, p := range ps {
		a = append(a, p.Name)
	}
	return a
}
func nativeSave() {
	if nativeCfg == nil {
		return
	}
	if v := nativeGetCombo(nativeBW); v != "" && !strings.HasPrefix(v, "—") {
		nativeCfg.PrinterBW = v
	}
	if v := nativeGetCombo(nativeColor); v != "" && !strings.HasPrefix(v, "—") {
		nativeCfg.PrinterColor = v
	}
	if v := nativeGetCombo(nativeP4); v != "" && !strings.HasPrefix(v, "—") {
		nativeCfg.Printer4x6 = v
	}
	if v := nativeGetCombo(nativeA3); v != "" && !strings.HasPrefix(v, "—") {
		nativeCfg.PrinterA3 = v
	}
	if v := nativeGetCombo(nativeDup); v != "" && !strings.HasPrefix(v, "—") {
		nativeCfg.PrinterDuplex = v
	}
	saveConfig(*nativeCfg)
	msg("QR Se Print", "Printer settings saved.", 0x40)
}
func nativeRefresh() {
	if nativeCfg == nil {
		return
	}
	ps := nativePrinterNames()
	nativeSetCombo(nativeBW, ps, nativeCfg.PrinterBW)
	nativeSetCombo(nativeColor, ps, nativeCfg.PrinterColor)
	nativeSetCombo(nativeP4, ps, nativeCfg.Printer4x6)
	nativeSetCombo(nativeA3, ps, nativeCfg.PrinterA3)
	nativeSetCombo(nativeDup, ps, nativeCfg.PrinterDuplex)
	if nativeClient != nil {
		reportPrinters(nativeClient, *nativeCfg)
	}
	if nativeHwnd != 0 {
		procInvalidateRect.Call(nativeHwnd, 0, 1)
	}
}
func nativeSyncNow() {
	if nativeCfg != nil && nativeClient != nil {
		heartbeat(nativeClient, *nativeCfg)
		reportPrinters(nativeClient, *nativeCfg)
		nativeUpdateStatus()
	}
}
func nativeOpenPlan() {
	procShellExecute.Call(0, uintptr(unsafe.Pointer(utf16("open"))), uintptr(unsafe.Pointer(utf16("https://bvv-djql.onrender.com/"))), 0, 0, swShow)
}
func nativeUpdateStatus() {
	stateMu.RLock()
	c := runtimeState.connected
	last := runtimeState.lastSync
	stateMu.RUnlock()
	if c {
		nativeSetText(nativeStatus, "● Print Agent Online")
		nativeSetText(nativeServer, "Yes")
	} else {
		nativeSetText(nativeStatus, "● Print Agent Offline")
		nativeSetText(nativeServer, "No")
	}
	nativeSetText(nativeSync, last)
}

func brush(rgb uint32) uintptr { r, _, _ := procCreateSolidBrush.Call(uintptr(rgb)); return r }
func paintText(hdc uintptr, text string, x, y, w, h int32, size int32, bold bool, rgb uint32) {
	weight := int32(400)
	if bold {
		weight = 700
	}
	font, _, _ := procCreateFont.Call(uintptr(-size), 0, 0, 0, uintptr(weight), 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(utf16("Segoe UI"))))
	old, _, _ := procSelectObject.Call(hdc, font)
	procSetBkMode.Call(hdc, 1)
	procSetTextColor.Call(hdc, uintptr(rgb))
	rc := wRect{x, y, x + w, y + h}
	procDrawText.Call(hdc, uintptr(unsafe.Pointer(utf16(text))), uintptr(len([]rune(text))), uintptr(unsafe.Pointer(&rc)), 0x0000)
	procSelectObject.Call(hdc, old)
	procDeleteObject.Call(font)
}
func fill(hdc uintptr, rc wRect, rgb uint32) {
	b := brush(rgb)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), b)
	procDeleteObject.Call(b)
}
func paintDashboard(hwnd uintptr, hdc uintptr) {
	fill(hdc, wRect{0, 0, 920, 780}, 0x00FFFFFF)
	// Header/logo
	fill(hdc, wRect{395, 20, 465, 90}, 0x00E91E63)
	paintText(hdc, "▦", 414, 30, 40, 45, 30, true, 0x00FFFFFF)
	paintText(hdc, "QR Se Print", 340, 98, 240, 40, 28, true, 0x00001B43)
	// cards
	fill(hdc, wRect{20, 150, 880, 430}, 0x00F9FAFD)
	fill(hdc, wRect{20, 150, 880, 430}, 0x00F9FAFD)
	fill(hdc, wRect{20, 150, 880, 430}, 0x00FFFFFF)
	for y := 190; y <= 390; y += 50 {
		fill(hdc, wRect{20, int32(y), 880, int32(y + 1)}, 0x00E7ECF4)
	}
	labels := []string{"Shop Name", "Shop ID", "Version", "Connected to Server", "Last Sync"}
	for i, l := range labels {
		paintText(hdc, l, 70, int32(174+i*50), 220, 30, 15, false, 0x0008285B)
	}
	// printer panel
	fill(hdc, wRect{20, 445, 880, 650}, 0x00FFF1F5)
	paintText(hdc, "▤", 38, 482, 35, 35, 22, true, 0x00E71964)
	paintText(hdc, "Select Printer", 75, 486, 160, 30, 17, true, 0x00001B43)
	paintText(hdc, "Black & White Printer", 210, 462, 250, 25, 13, true, 0x007A425A)
	paintText(hdc, "Color Printer", 525, 462, 220, 25, 13, true, 0x007A425A)
	paintText(hdc, "4×6 Photo Printer", 210, 550, 220, 25, 13, true, 0x007A425A)
	paintText(hdc, "A3 Printer", 525, 550, 180, 25, 13, true, 0x007A425A)
	paintText(hdc, "Duplex Printer", 210, 600, 220, 25, 13, true, 0x007A425A)
	// button backgrounds are drawn by owner-draw buttons
}
func buttonColor(id int) uint32 {
	switch id {
	case idSave:
		return 0x00E71964
	case idSync:
		return 0x0013A34A
	case idRefresh:
		return 0x004036CF
	case idPlan:
		return 0x00F39A00
	default:
		return 0x00505A6E
	}
}
func drawButton(di *drawItem) {
	c := buttonColor(int(di.CtlID))
	fill(di.hdc, di.rcItem, c)
	label := ""
	switch int(di.CtlID) {
	case idSave:
		label = "▣  Save Printer"
	case idSync:
		label = "⟳  Sync Now"
	case idRefresh:
		label = "⟳  Refresh Printer List"
	case idPlan:
		label = "⚡  Demo se Paid Shop par jao"
	case idCounter:
		if runtimeCounterApproval() {
			label = "🔔  Counter Approval: ON"
		} else {
			label = "🔔  Counter Approval: OFF"
		}
	case idReconnect:
		label = "↻  Reconnect"
	case idChangeShop:
		label = "▣  Change Shop ID"
	case idLogs:
		label = "▤  View Logs"
	}
	paintText(di.hdc, label, di.rcItem.left+10, di.rcItem.top+7, di.rcItem.right-di.rcItem.left-20, di.rcItem.bottom-di.rcItem.top-12, 14, true, 0x00FFFFFF)
}

func nativeWndProc(hwnd uintptr, msgid uint32, wParam, lParam uintptr) uintptr {
	switch msgid {
	case wmPaint:
		var ps wPaint
		r, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		paintDashboard(hwnd, r)
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmDrawItem:
		di := (*drawItem)(unsafe.Pointer(lParam))
		drawButton(di)
		return 1
	case wmCommand:
		id := int(wParam & 0xffff)
		code := int((wParam >> 16) & 0xffff)
		if code == bnClicked {
			switch id {
			case idSave:
				nativeSave()
			case idSync:
				nativeSyncNow()
			case idRefresh:
				nativeRefresh()
			case idPlan:
				nativeOpenPlan()
			case idReconnect:
				nativeSyncNow()
			case idCounter:
				stateMu.Lock()
				runtimeState.counterApproval = !runtimeState.counterApproval
				stateMu.Unlock()
				procInvalidateRect.Call(hwnd, 0, 1)
			case idChangeShop:
				if nativeCfg != nil {
					next := doLogin(*nativeCfg)
					if next.AgentToken != "" {
						*nativeCfg = next
						nativeSetText(nativeShop, next.ShopID)
						nativeSetText(nativeVersion, "v"+next.Version)
						nativeSyncNow()
						nativeRefresh()
						showDemoUpgradePrompt(*nativeCfg)
					}
				}
			case idLogs:
				_ = exec.Command("notepad.exe", filepath.Join(filepath.Dir(configPath()), "agent.log")).Start()
			}
		}
		return 0
	case wmClose:
		procShowWindow.Call(hwnd, swHide)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msgid), wParam, lParam)
	return r
}

func startNativeDashboard(cfg *Config, client *http.Client) {
	runtime.LockOSThread()
	nativeCfg = cfg
	nativeClient = client
	inst, _, _ := procGetModuleHandle.Call(0)
	className := utf16("QRSePrintNativeDashboardV2")
	wc := wWndClassEx{cbSize: uint32(unsafe.Sizeof(wWndClassEx{})), wndProc: syscall.NewCallback(nativeWndProc), hInstance: inst, className: className}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd := nativeCreate(0, "QRSePrintNativeDashboardV2", "QR Se Print — Print Agent", wsOverlapped, 0, 0, 920, 760, 0)
	nativeHwnd = hwnd
	nativeLabel(hwnd, "Shop Name", 70, 174, 220, 30)
	nativeShop = nativeLabel(hwnd, cfg.ShopID, 600, 174, 240, 30)
	nativeLabel(hwnd, "Shop ID", 70, 224, 220, 30)
	nativeLabel(hwnd, cfg.ShopID, 600, 224, 240, 30)
	nativeLabel(hwnd, "Version", 70, 274, 220, 30)
	nativeVersion = nativeLabel(hwnd, "v"+cfg.Version, 600, 274, 240, 30)
	nativeLabel(hwnd, "Connected to Server", 70, 324, 260, 30)
	nativeServer = nativeLabel(hwnd, "...", 740, 324, 80, 30)
	nativeLabel(hwnd, "Last Sync", 70, 374, 220, 30)
	nativeSync = nativeLabel(hwnd, "—", 740, 374, 80, 30)
	nativeStatus = nativeLabel(hwnd, "● Connecting...", 350, 130, 250, 30)
	nativeBW = nativeCombo(hwnd, 210, 492, 300, 38, idBW)
	nativeColor = nativeCombo(hwnd, 525, 492, 300, 38, idColor)
	nativeP4 = nativeCombo(hwnd, 210, 575, 300, 38, idP4)
	nativeA3 = nativeCombo(hwnd, 525, 575, 300, 38, idA3)
	nativeDup = nativeCombo(hwnd, 210, 625, 300, 38, idDup)
	nativeButton(hwnd, "Save Printer", 20, 665, 250, 50, idSave)
	nativeButton(hwnd, "Sync Now", 285, 665, 250, 50, idSync)
	nativeButton(hwnd, "Refresh Printer List", 550, 665, 330, 50, idRefresh)
	nativeCounterButton = nativeButton(hwnd, "Counter Approval: ON", 20, 720, 200, 35, idCounter)
	nativeButton(hwnd, "Reconnect", 230, 720, 160, 35, idReconnect)
	nativeButton(hwnd, "Change Shop ID", 400, 720, 190, 35, idChangeShop)
	nativeButton(hwnd, "View Logs", 600, 720, 160, 35, idLogs)
	nativeRefresh()
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	go func() {
		for {
			time.Sleep(2 * time.Second)
			if nativeHwnd != 0 {
				nativeUpdateStatus()
			}
		}
	}()
	var m wMsg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func nativeLoginDialog(server, shop string) loginResult {
	runtime.LockOSThread()
	ls := &loginState{result: loginResult{server: server, shop: shop}}
	inst, _, _ := procGetModuleHandle.Call(0)
	cn := utf16("QRSePrintNativeLogin")
	wc := wWndClassEx{cbSize: uint32(unsafe.Sizeof(wWndClassEx{})), wndProc: syscall.NewCallback(func(hwnd uintptr, m uint32, w, l uintptr) uintptr {
		switch m {
		case wmPaint:
			var ps wPaint
			r, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
			fill(r, wRect{0, 0, 520, 330}, 0x00FFFFFF)
			paintText(r, "QR Se Print", 28, 25, 350, 35, 24, true, 0x00001B43)
			paintText(r, "Setup / Login", 28, 70, 300, 28, 15, false, 0x00607085)
			procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
			return 0
		case wmCommand:
			id := int(w & 0xffff)
			if int((w>>16)&0xffff) == bnClicked {
				if id == idLogin {
					read := func(h uintptr) string {
						b := make([]uint16, 512)
						procSendMessage.Call(h, 0x000D, 511, uintptr(unsafe.Pointer(&b[0])))
						return syscall.UTF16ToString(b)
					}
					ls.result.shop = read(ls.editShop)
					ls.result.password = read(ls.editPass)
					ls.result.server = read(ls.editServer)
					ls.result.cancelled = false
					procDestroyWindow.Call(hwnd)
					return 0
				}
				if id == idCancelLogin {
					ls.result.cancelled = true
					procDestroyWindow.Call(hwnd)
					return 0
				}
			}
		case wmClose:
			ls.result.cancelled = true
			procDestroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			procPostQuitMessage.Call(0)
			return 0
		}
		r, _, _ := procDefWindowProc.Call(hwnd, uintptr(m), w, l)
		return r
	}), hInstance: inst, className: cn}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd := nativeCreate(0, "QRSePrintNativeLogin", "QR Se Print - Setup", wsOverlapped, 0, 0, 520, 330, 0)
	ls.hwnd = hwnd
	nativeLabel(hwnd, "Server", 28, 115, 100, 25)
	ls.editServer = nativeEdit(hwnd, server, 130, 110, 350, 30, idLoginServer, false)
	nativeLabel(hwnd, "Shop ID", 28, 155, 100, 25)
	ls.editShop = nativeEdit(hwnd, shop, 130, 150, 350, 30, idLoginShop, false)
	nativeLabel(hwnd, "Password", 28, 195, 100, 25)
	ls.editPass = nativeEdit(hwnd, "", 130, 190, 350, 30, idLoginPass, true)
	ls.btnLogin = nativeButton(hwnd, "Login", 130, 245, 150, 45, idLogin)
	ls.btnCancel = nativeButton(hwnd, "Cancel", 300, 245, 150, 45, idCancelLogin)
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)
	procSetFocus.Call(ls.editShop)
	var m wMsg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	return ls.result
}

func launchTray() { go nativeTrayLoop() }
func nativeTrayLoop() {
	runtime.LockOSThread()
	inst, _, _ := procGetModuleHandle.Call(0)
	cn := utf16("QRSePrintNativeTray")
	wndProc := syscall.NewCallback(func(hwnd uintptr, m uint32, w, l uintptr) uintptr {
		if m == trayMsg {
			if l == 0x0203 || l == 0x0201 {
				if nativeHwnd != 0 {
					procShowWindow.Call(nativeHwnd, swShow)
				}
			}
			if l == 0x0205 {
				menu, _, _ := procCreatePopupMenu.Call()
				procAppendMenu.Call(menu, 0, trayOpen, uintptr(unsafe.Pointer(utf16("Open Dashboard"))))
				procAppendMenu.Call(menu, 0, traySync, uintptr(unsafe.Pointer(utf16("Sync Now"))))
				procAppendMenu.Call(menu, 0, trayReconnect, uintptr(unsafe.Pointer(utf16("Reconnect to Server"))))
				procAppendMenu.Call(menu, 0, trayExit, uintptr(unsafe.Pointer(utf16("Exit"))))
				var pt struct{ x, y int32 }
				procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
				procSetForegroundWindow.Call(hwnd)
				cmd, _, _ := procTrackPopupMenu.Call(menu, 0x0100, uintptr(pt.x), uintptr(pt.y), 0, hwnd, 0)
				procDestroyMenu.Call(menu)
				switch cmd {
				case trayOpen:
					if nativeHwnd != 0 {
						procShowWindow.Call(nativeHwnd, swShow)
					}
				case traySync:
					nativeSyncNow()
				case trayReconnect:
					nativeSyncNow()
				case trayExit:
					os.Exit(0)
				}
			}
			return 0
		}
		if m == wmCommand {
			switch w & 0xffff {
			case trayOpen:
				if nativeHwnd != 0 {
					procShowWindow.Call(nativeHwnd, swShow)
				}
			case traySync:
				nativeSyncNow()
			case trayReconnect:
				nativeSyncNow()
			case trayExit:
				os.Exit(0)
			}
		}
		if m == wmDestroy {
			procPostQuitMessage.Call(0)
			return 0
		}
		r, _, _ := procDefWindowProc.Call(hwnd, uintptr(m), w, l)
		return r
	})
	wc := wWndClassEx{cbSize: uint32(unsafe.Sizeof(wWndClassEx{})), wndProc: wndProc, hInstance: inst, className: cn}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd := nativeCreate(0, "QRSePrintNativeTray", "", 0, 0, 0, 0, 0, 0)
	var nid trayIconData
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hwnd = hwnd
	nid.uID = trayUID
	nid.uFlags = 1 | 2 | 4
	nid.uCallbackMessage = trayMsg
	tip, _ := syscall.UTF16FromString("QR Se Print Agent")
	copy(nid.tip[:], tip)
	procShellNotifyIcon.Call(0x00000000, uintptr(unsafe.Pointer(&nid)))
	var m wMsg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
