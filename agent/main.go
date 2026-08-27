//go:build windows
package main

import (
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "syscall"
 "time"
 "unsafe"
)

type Config struct { ServerURL, ShopID, AgentToken, Printer string; PollSeconds int }
type Job struct { ID string `json:"id"`; Status string `json:"status"`; FileURL string `json:"file_url"`; FileName string `json:"file_name"`; Copies int `json:"copies"` }
type PollResp struct { Success bool `json:"success"`; Job *Job `json:"job"`; Error string `json:"error"` }
var user32=syscall.NewLazyDLL("user32.dll"); var messageBox=user32.NewProc("MessageBoxW")
func msg(title,text string,flags uintptr) int { t,_:=syscall.UTF16PtrFromString(text); c,_:=syscall.UTF16PtrFromString(title); r,_,_:=messageBox.Call(0,uintptr(unsafe.Pointer(t)),uintptr(unsafe.Pointer(c)),flags); return int(r) }
func main(){
 cfg:=Config{ServerURL:"http://localhost:8080",ShopID:"DEMO",AgentToken:"agent_demo",PollSeconds:3}
 if b,e:=os.ReadFile("agent-config.json");e==nil{_ = json.Unmarshal(b,&cfg)}
 if v:=os.Getenv("QRSEPRINT_SERVER");v!=""{cfg.ServerURL=v}; if v:=os.Getenv("QRSEPRINT_SHOP");v!=""{cfg.ShopID=v}; if v:=os.Getenv("QRSEPRINT_TOKEN");v!=""{cfg.AgentToken=v}
 if cfg.PollSeconds<1{cfg.PollSeconds=3}
 msg("QR Se Print Agent","Agent started for shop: "+cfg.ShopID,0x40)
 client:=&http.Client{Timeout:30*time.Second}
 seen:=map[string]bool{}
 for { j,e:=poll(client,cfg); if e==nil && j!=nil && !seen[j.ID] { seen[j.ID]=true; handle(client,cfg,*j) }; time.Sleep(time.Duration(cfg.PollSeconds)*time.Second) }
}
func poll(c *http.Client,cfg Config)(*Job,error){
 u:=strings.TrimRight(cfg.ServerURL,"/")+"/api/agent/poll?shopId="+url.QueryEscape(cfg.ShopID)+"&token="+url.QueryEscape(cfg.AgentToken)
 r,e:=c.Get(u);if e!=nil{return nil,e};defer r.Body.Close();if r.StatusCode!=200{return nil,fmt.Errorf("poll http %d",r.StatusCode)};var x PollResp;e=json.NewDecoder(r.Body).Decode(&x);if e!=nil{return nil,e};if !x.Success{return nil,fmt.Errorf(x.Error)};return x.Job,nil
}
func handle(c *http.Client,cfg Config,j Job){
 text:=fmt.Sprintf("New print order\n\nOrder: %s\nFile: %s\nCopies: %d\n\nAccept this print job?",j.ID,j.FileName,j.Copies)
 r:=msg("QR Se Print — Print Request",text,0x24|0x1000) // MB_YESNO | MB_ICONQUESTION
 if r!=6 { _=postJSON(c,cfg,"/api/agent/reject",map[string]any{"shopId":cfg.ShopID,"token":cfg.AgentToken,"jobId":j.ID,"error":"Rejected by operator"}); return }
 var claim struct{Success bool `json:"success"`; FileURL string `json:"fileUrl"`; Error string `json:"error"`}
 if e:=postJSON(c,cfg,"/api/agent/claim",map[string]any{"shopId":cfg.ShopID,"token":cfg.AgentToken,"jobId":j.ID},&claim);e!=nil||!claim.Success{msg("QR Se Print","Could not accept order: "+fmt.Sprint(e),0x10);return}
 fileURL:=claim.FileURL;if strings.HasPrefix(fileURL,"/"){fileURL=strings.TrimRight(cfg.ServerURL,"/")+fileURL}
 local:=filepath.Join(os.TempDir(),"QRSePrint",j.ID+filepath.Ext(j.FileName));_ = os.MkdirAll(filepath.Dir(local),0755)
 if e:=download(c,fileURL,local);e!=nil{finish(c,cfg,j.ID,false,e.Error());msg("QR Se Print","Download failed: "+e.Error(),0x10);return}
 copies:=j.Copies;if copies<1{copies=1};if cfg.Printer!=""{_ = setDefaultPrinter(cfg.Printer)}
 var pe error
 for i:=0;i<copies;i++ {if e:=printFile(local);e!=nil{pe=e;break};time.Sleep(1200*time.Millisecond)}
 if pe!=nil {finish(c,cfg,j.ID,false,pe.Error());msg("QR Se Print","Print failed: "+pe.Error(),0x10)} else {finish(c,cfg,j.ID,true,"");msg("QR Se Print","Order "+j.ID+" printed successfully.",0x40)}
}
func download(c *http.Client,u,p string)error{r,e:=c.Get(u);if e!=nil{return e};defer r.Body.Close();if r.StatusCode!=200{return fmt.Errorf("download http %d",r.StatusCode)};f,e:=os.Create(p);if e!=nil{return e};defer f.Close();_,e=io.Copy(f,r.Body);return e}
func printFile(p string)error{cmd:=exec.Command("powershell.exe","-NoProfile","-NonInteractive","-Command","Start-Process -LiteralPath $args[0] -Verb Print -Wait",p);return cmd.Run()}
func setDefaultPrinter(name string)error{return exec.Command("rundll32.exe","printui.dll,PrintUIEntry","/y","/n",name).Run()}
func finish(c *http.Client,cfg Config,id string,ok bool,err string){_=postJSON(c,cfg,"/api/agent/complete",map[string]any{"shopId":cfg.ShopID,"token":cfg.AgentToken,"jobId":id,"success":ok,"error":err},nil)}
func postJSON(c *http.Client,cfg Config,route string,v any,out ...any)error{b,_:=json.Marshal(v);r,e:=c.Post(strings.TrimRight(cfg.ServerURL,"/")+route,"application/json",strings.NewReader(string(b)));if e!=nil{return e};defer r.Body.Close();if r.StatusCode>=300{return fmt.Errorf("http %d",r.StatusCode)};if len(out)>0&&out[0]!=nil{return json.NewDecoder(r.Body).Decode(out[0])};return nil}
var _ = strconv.Itoa
