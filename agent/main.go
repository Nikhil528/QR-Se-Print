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

//go:embed ui.html
var uiHTML []byte

// Embed helper scripts so the EXE works even when it is copied/run by itself.
// The scripts are extracted beside agent-config.json on first launch.
//go:embed login.ps1
var loginPS1 []byte

//go:embed tray.ps1
var trayPS1 []byte

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
	startLocalUI(&cfg, client)
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

func ensureEmbeddedScripts() {
	dir := filepath.Dir(configPath())
	_ = os.MkdirAll(dir, 0700)
	loginPath := filepath.Join(dir, "login.ps1")
	if _, err := os.Stat(loginPath); err != nil { _ = os.WriteFile(loginPath, loginPS1, 0600) }
	trayPath := filepath.Join(dir, "tray.ps1")
	if _, err := os.Stat(trayPath); err != nil { _ = os.WriteFile(trayPath, trayPS1, 0600) }
}

func doLogin(cfg Config) Config {
	ensureEmbeddedScripts()
	out, err := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(filepath.Dir(configPath()), "login.ps1"), "-DefaultServer", cfg.ServerURL, "-DefaultShop", cfg.ShopID).Output()
	if err != nil {
		msg("QR Se Print", "Login window could not open: "+err.Error(), 0x10)
		return cfg
	}
	vals := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if i := strings.IndexByte(line, '='); i > 0 {
			vals[line[:i]] = strings.TrimSpace(line[i+1:])
		}
	}
	server := vals["SERVER"]
	shop := vals["SHOP"]
	pass := vals["PASSWORD"]
	if server == "" || shop == "" || pass == "" {
		return cfg
	}
	client := &http.Client{Timeout: 30 * time.Second}
	var lr LoginResponse
	if err := postJSON(client, server, "/api/agent/login", map[string]any{"shopId": shop, "password": pass}, &lr); err != nil || !lr.Success {
		msg("QR Se Print", "Login failed: "+firstErr(err, lr.Error), 0x10)
		return cfg
	}
	cfg.ServerURL = strings.TrimRight(server, "/")
	cfg.ShopID = lr.ShopID
	cfg.AgentToken = lr.AgentToken
	cfg.AgentID = lr.AgentID
	cfg.Version = "9.1.0"
	saveConfig(cfg)
	msg("QR Se Print", "Login successful\n\nShop: "+cfg.ShopID+"\nAgent is ready for print jobs.", 0x40)
	return cfg
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
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-Printer | Select-Object Name,PrinterStatus | ConvertTo-Json -Compress").Output()
	if err != nil {
		return nil
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil
	}
	var one Printer
	var many []Printer
	if strings.HasPrefix(s, "[") {
		_ = json.Unmarshal(out, &many)
	} else {
		_ = json.Unmarshal(out, &one)
		many = []Printer{one}
	}
	return many
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
	// Return -1 only when PowerShell itself fails. Do not use -1 as a real queue count.
	script := `$p = $env:QRSE_PRINTER; try { @(Get-PrintJob -PrinterName $p -ErrorAction Stop).Count } catch { exit 2 }`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "QRSE_PRINTER="+name)
	out, err := cmd.Output()
	if err != nil {
		return -1
	}
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &n); err != nil {
		return -1
	}
	return n
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

func startLocalUI(cfg *Config, client *http.Client) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(uiHTML)
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		stateMu.RLock()
		rs := runtimeState
		jobs := append([]map[string]any(nil), rs.jobs...)
		stateMu.RUnlock()
		ps := enumeratePrinters()
		names := make([]string, 0, len(ps))
		for _, p := range ps {
			names = append(names, p.Name)
		}
		stateMu.RLock()
		connected := runtimeState.connected
		last := runtimeState.lastSync
		stateMu.RUnlock()
		stateMu.RLock()
		ca := runtimeState.counterApproval
		stateMu.RUnlock()
		out := map[string]any{"connected": connected, "lastSync": last, "printers": names, "config": cfg, "jobs": jobs, "counterApproval": ca, "shopName": cfg.ShopID, "demo": strings.HasPrefix(strings.ToUpper(cfg.ShopID), "DEMO_"), "demoMessage": "Aap DEMO Shop ID use kar rahe hain. Paid plan ke liye upgrade karein."}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		var x map[string]string
		if json.NewDecoder(r.Body).Decode(&x) == nil {
			if v := strings.TrimSpace(x["bw"]); v != "" {
				cfg.PrinterBW = v
			}
			if v := strings.TrimSpace(x["color"]); v != "" {
				cfg.PrinterColor = v
			}
			if v := strings.TrimSpace(x["p4"]); v != "" {
				cfg.Printer4x6 = v
			}
			if v := strings.TrimSpace(x["a3"]); v != "" {
				cfg.PrinterA3 = v
			}
			if v := strings.TrimSpace(x["dup"]); v != "" {
				cfg.PrinterDuplex = v
			}
			saveConfig(*cfg)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		heartbeat(client, *cfg)
		reportPrinters(client, *cfg)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/reconnect", func(w http.ResponseWriter, r *http.Request) {
		heartbeat(client, *cfg)
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
	mux.HandleFunc("/plan", func(w http.ResponseWriter, r *http.Request) {
		_ = exec.Command("cmd", "/c", "start", "", "https://bvv-djql.onrender.com/").Start()
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		_ = exec.Command("notepad.exe", filepath.Join(filepath.Dir(configPath()), "agent.log")).Start()
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/open", func(w http.ResponseWriter, r *http.Request) {
		_ = exec.Command("cmd", "/c", "start", "", "http://127.0.0.1:17845/").Start()
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/change-shop", func(w http.ResponseWriter, r *http.Request) {
		next := doLogin(*cfg)
		if next.AgentToken != "" {
			*cfg = next
			heartbeat(client, *cfg)
			reportPrinters(client, *cfg)
			showDemoUpgradePrompt(*cfg)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	go func() { _ = http.ListenAndServe("127.0.0.1:17845", mux) }()
}

func ensureAutoStart() {
	exe, _ := os.Executable()
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	cmd := `reg add "` + key + `" /v "QR Se Print Agent" /t REG_SZ /d """` + exe + `""" /f`
	_ = exec.Command("cmd", "/c", cmd).Run()
	logLine("Auto-start ensured via HKCU Run")
}

func launchTray() {
	ensureEmbeddedScripts()
	ps := filepath.Join(filepath.Dir(configPath()), "tray.ps1")
	if _, e := os.Stat(ps); e != nil {
		return
	}
	_ = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-File", ps).Start()
}
