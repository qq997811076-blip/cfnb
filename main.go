// Cloudflare IP 优选工具 - Go 迁移版
// 只使用 Go 标准库；读取原有 config.json，保留原参数名/值。
// 带宽测速继续调用原配置所用的 curl；GitHub 同步继续使用原 git_sync.sh / git_sync.ps1。
package main

import (
 "bufio"
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "sort"
 "strconv"
 "strings"
 "sync"
 "sync/atomic"
 "time"
)

type Config map[string]any
type NodeResult struct { Node string; Latency float64; Country string; Success int }
type AvailabilityResult struct { Node string; OK bool; Stack string; Exit map[string]any }
type HTTPResult struct { Node string; OK bool; Server string; Latency float64; Jitter float64 }
type BWResult struct { Node string; Speed float64 }
type Source struct { URL string `json:"url"`; Enabled bool `json:"enabled"` }

var cfg Config
var root string
var cnToCode = map[string]string{

}
var alpha3ToAlpha2 = map[string]string{

}
var codeSet = map[string]bool{}
var nodePattern = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#(.+)$`)
var ipPortPattern = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#`)
var muLog sync.Mutex

func val(k string) any { return cfg[k] }
func str(k string) string { if v,ok:=cfg[k].(string); ok{return v}; return fmt.Sprint(cfg[k]) }
func boolean(k string) bool { v,ok:=cfg[k].(bool); return ok && v }
func number(k string) float64 { switch v:=cfg[k].(type) { case float64:return v; case int:return float64(v); case int64:return float64(v); case json.Number:f,_:=v.Float64();return f }; return 0 }
func integer(k string) int { return int(number(k)) }
func stringsVal(k string) []string { v,ok:=cfg[k].([]any); if !ok { if s,ok2:=cfg[k].([]string);ok2{return s}; return nil }; out:=make([]string,0,len(v)); for _,x:=range v {out=append(out,fmt.Sprint(x))};return out }
func duration(k string, fallback float64) time.Duration { n:=number(k); if n<=0 {n=fallback}; return time.Duration(n*float64(time.Second)) }
func logf(format string,args ...any) { muLog.Lock(); defer muLog.Unlock(); line:=fmt.Sprintf(format,args...); fmt.Print(line); if boolean("ENABLE_LOGGING") { f,e:=os.OpenFile(filepath.Join(root,str("LOG_FILE")),os.O_CREATE|os.O_APPEND|os.O_WRONLY,0644); if e==nil { _,_=f.WriteString(line); f.Close() } } }
func loadConfig() error {
 p:=filepath.Join(root,"config.json")
 b,e:=os.ReadFile(p); if e!=nil {return fmt.Errorf("读取 config.json 失败: %w",e)}
 if e=json.Unmarshal(b,&cfg);e!=nil{return fmt.Errorf("config.json 格式错误: %w",e)}
 // 保留原配置中的所有键和所有值；只为缺少的键补上与原脚本相同的默认值。
 defaults:=map[string]any{
 "USE_GLOBAL_MODE":true,"GLOBAL_TOP_N":15,"PER_COUNTRY_TOP_N":1,"BANDWIDTH_CANDIDATES":150,"TCP_PROBES":1,"MIN_SUCCESS_RATE":1.0,"TCP_LATENCY_WEIGHT":0.0,"TIMEOUT":2.0,"SOCKET_DEFAULT_TIMEOUT":3,"PROGRESS_PRINT_INTERVAL":1,
 "FILTER_COUNTRIES_ENABLED":false,"ALLOWED_COUNTRIES":[]any{"US"},"PRE_FILTER_BLOCKED_ENABLED":true,"PRE_FILTER_BLOCKED_COUNTRIES":[]any{"CN"},"PRE_FILTER_PORT_ENABLED":true,"PRE_FILTER_PORTS":[]any{443},
 "ENABLE_WXPUSHER":true,"WXPUSHER_APP_TOKEN":"your_app_token_here","WXPUSHER_UIDS":[]any{"your_uid_here"},"WXPUSHER_API_URL":"https://wxpusher.zjiecode.com/api/send/message","NOTIFY_TIMEOUT":3,"NOTIFY_CONNECT_TIMEOUT":3,
 "CF_ENABLED":true,"CF_API_TOKEN":"your_CF_API_TOKEN","CF_ZONE_ID":"your_CF_ZONE_ID","CF_DNS_RECORD_NAME":"your_CF_DNS_RECORD_NAME","CF_TTL":60,"CF_PROXIED":false,"CF_DNS_CONNECT_TIMEOUT":3,"CF_DNS_READ_TIMEOUT":3,"DNS_RECORD_TYPE":"TXT",
 "ADDITIONAL_SOURCES":[]any{},"FETCH_MAX_RETRIES":3,"FETCH_RETRY_DELAY":3,"FETCH_TIMEOUT":3,"FETCH_CONNECT_TIMEOUT":3,"IP_CALIBRATION_ENABLED":false,"TOKEN_FAILURE_THRESHOLD":3,"IP_CALIBRATION_MIN_INTERVAL":0.1,"IP_CALIBRATION_TOKEN_FILE":"valid_tokens.txt","IP_CALIBRATION_CACHE_FILE":"ipinfo_cache.txt","OUTPUT_FILE":"ip.txt","ENABLE_LOGGING":false,"LOG_FILE":"cfnb.log","FORCE_DIRECT":false,
 "TEST_AVAILABILITY":true,"AVAILABILITY_CHECK_API":"https://api.090227.xyz/check","AVAILABILITY_TIMEOUT":3,"AVAILABILITY_CONNECT_TIMEOUT":3,"AVAILABILITY_RETRY_MAX":2,"AVAILABILITY_RETRY_DELAY":3,"AVAILABILITY_INNER_RETRY_ENABLED":true,"AVAILABILITY_INNER_RETRY_MAX":2,"AVAILABILITY_INNER_RETRY_DELAY":3,
 "HTTP_TEST_ENABLED":true,"HTTP_TEST_TIMEOUT":3,"HTTP_TEST_CONNECT_TIMEOUT":3,"HTTP_TEST_MAX_ROUNDS":2,"HTTP_TEST_ROUND_DELAY":3,"HTTP_TEST_INNER_RETRY_ENABLED":true,"HTTP_TEST_MAX_RETRIES":2,"HTTP_TEST_RETRY_DELAY":3,"HTTP_TEST_METHOD":"HEAD","HTTP_LATENCY_WEIGHT":3.0,"JITTER_WEIGHT":3.0,"HTTP_JITTER_SAMPLES":3,
 "FILTER_IPV6_AVAILABILITY":true,"FILTER_BLOCKED_COUNTRIES_ENABLED":true,"BLOCKED_COUNTRIES":[]any{"BD","BI","BY","CD","CF","CN","CU","DE","ET","HK","IR","KP","LY","MO","NG","NL","PK","RU","SD","SO","SY","TH","TW","UA","VE","VN","YE","ZW"},
 "DNS_IP_RISK_FILTER_ENABLED":false,"DNS_IP_RISK_MAX_LEVEL":"高风险","DNS_UPDATE_TARGET_COUNT":15,"BANDWIDTH_SIZE_MB":1.0,"BANDWIDTH_TIMEOUT":3,"BANDWIDTH_RETRY_MAX":2,"BANDWIDTH_RETRY_DELAY":3,"BANDWIDTH_URL_TEMPLATE":"{scheme}://speed.cloudflare.com:{port}/__down?bytes={bytes}","BANDWIDTH_PROCESS_BUFFER":2,"BANDWIDTH_CONNECT_TIMEOUT":3,"SPEED_WEIGHT":3.0,
 "IP_CALIBRATION_CONCURRENCY":300,"MAX_WORKERS":300,"AVAILABILITY_WORKERS":32,"FALLBACK_WORKERS":32,"BANDWIDTH_WORKERS":3,"HTTP_TEST_WORKERS":32,"DNS_UPDATE_MAX_RETRIES":3,"DNS_UPDATE_RETRY_DELAY":3,"GITHUB_SYNC_MAX_RETRIES":3,"GITHUB_SYNC_RETRY_DELAY":3,"GIT_SYNC_PROCESS_TIMEOUT":180,
 "AD_HEADER_ENABLED":false,"AD_HEADER_LINES":[]any{},"AD_FOOTER_ENABLED":false,"AD_FOOTER_LINES":[]any{},"AD_PERLINE_ENABLED":false,"AD_PERLINE_TEXT":"","IP_TXT_SHOW_BANDWIDTH":false,"IP_TXT_SHOW_HTTP_LATENCY":false,"IP_TXT_SHOW_HTTP_JITTER":false,"IP_TXT_SHOW_LATENCY":false,
 }
 for k,v:=range defaults {if _,ok:=cfg[k];!ok {cfg[k]=v;logf("配置项 %s 未设置，使用默认值：%v\n",k,v)}}
 for c:=range cnToCode { _=c }; for _,c:=range cnToCode {codeSet[c]=true}
 return nil
}
func newHTTPClient(connect, read time.Duration) *http.Client {
 tr:=&http.Transport{DialContext:(&net.Dialer{Timeout:connect,KeepAlive:30*time.Second}).DialContext, TLSHandshakeTimeout:connect, ResponseHeaderTimeout:read, IdleConnTimeout:30*time.Second, MaxIdleConns:128, MaxIdleConnsPerHost:64}
 if !boolean("FORCE_DIRECT") { tr.Proxy=http.ProxyFromEnvironment }
 return &http.Client{Transport:tr,Timeout:connect+read+2*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{return http.ErrUseLastResponse}}
}
func request(method, target string, headers map[string]string, body []byte, connect, read time.Duration) (*http.Response,[]byte,error) {
 req,e:=http.NewRequest(method,target,bytes.NewReader(body));if e!=nil{return nil,nil,e}
 for k,v:=range headers {req.Header.Set(k,v)}
 resp,e:=newHTTPClient(connect,read).Do(req);if e!=nil{return nil,nil,e}
 b,re:=io.ReadAll(resp.Body);resp.Body.Close();if re!=nil{return resp,b,re};return resp,b,nil
}
func sleepSec(k string) { n:=number(k); if n>0 {time.Sleep(time.Duration(n*float64(time.Second)))} }
func notify(content,summary string) {
 if !boolean("ENABLE_WXPUSHER"){return}
 payload:=map[string]any{"appToken":str("WXPUSHER_APP_TOKEN"),"content":content,"summary":summary,"uids":val("WXPUSHER_UIDS")}
 b,_:=json.Marshal(payload)
 resp,_,e:=request("POST",str("WXPUSHER_API_URL"),map[string]string{"Content-Type":"application/json; charset=utf-8"},b,duration("NOTIFY_CONNECT_TIMEOUT",3),duration("NOTIFY_TIMEOUT",3))
 if e!=nil {logf("微信通知异常: %v\n",e)} else if resp.StatusCode==200 {logf("微信通知已发送\n")} else {logf("微信通知发送失败: %d\n",resp.StatusCode)}
}
func extractCountryCode(label string) string {
 label=strings.TrimSpace(label); if label=="" {return ""}
 re:=regexp.MustCompile(`[\s,;|/]+`)
 for _,tok:=range re.Split(label,-1) {
  tok=regexp.MustCompile(`^[\d\s\-_.|#]+`).ReplaceAllString(strings.TrimSpace(tok),"")
  m:=regexp.MustCompile(`^([A-Z]{3})(?![A-Za-z])`).FindStringSubmatch(tok)
  if len(m)>1 {if c:=alpha3ToAlpha2[m[1]];c!=""{return c}}
  m=regexp.MustCompile(`^([A-Z]{2})(?![A-Za-z])`).FindStringSubmatch(tok)
  if len(m)>1 && codeSet[m[1]] {return m[1]}
 }
 for _,tok:=range re.Split(label,-1) {
  tok=regexp.MustCompile(`^[\d\s\-_.|#]+`).ReplaceAllString(tok,"")
  for name,code:=range cnToCode {if strings.Contains(tok,name){return code}}
 }
 // Regional indicator flag emoji (first two indicators)
 var indicators []rune; for _,r:=range label {if r>=0x1F1E6 && r<=0x1F1FF {indicators=append(indicators,r)}}
 if len(indicators)>=2 {return string([]rune{rune('A')+indicators[0]-0x1F1E6,rune('A')+indicators[1]-0x1F1E6})}
 return ""
}
func queryCountry(ip,port string) string {
 u,_:=url.Parse(str("AVAILABILITY_CHECK_API")); q:=u.Query();q.Set("proxyip",ip+":"+port);u.RawQuery=q.Encode()
 resp,b,e:=request("GET",u.String(),nil,nil,duration("AVAILABILITY_CONNECT_TIMEOUT",3),duration("AVAILABILITY_TIMEOUT",3))
 if e!=nil || resp.StatusCode!=200{return ""}
 var d map[string]any;_ = json.Unmarshal(b,&d)
 country:=nestedString(d,"probe_results","ipv4","exit","country"); if len(country)==2{return strings.ToUpper(country)};return ""
}
func nestedString(m map[string]any, keys ...string) string {var x any=m;for _,k:=range keys {mm,ok:=x.(map[string]any);if !ok{return ""};x=mm[k]};s,_:=x.(string);return s}
func parseJSONNodes(x any) []string {
 var out []string
 switch v:=x.(type) {
 case []any: for _,it:=range v {out=append(out,parseJSONNodes(it)...)}
 case map[string]any:
  for _,k:=range []string{"nodes","data","result","list"} {if a,ok:=v[k].([]any);ok {out=append(out,parseJSONNodes(a)...);break}}
  ip:=fmt.Sprint(v["ip"]);if ip=="<nil>"||ip=="" {ip=fmt.Sprint(v["host"])}
  port:=fmt.Sprint(v["port"]);cc:=fmt.Sprint(v["country"]);if cc=="<nil>"||cc==""{cc=fmt.Sprint(v["cc"])}
  if ip!=""&&ip!="<nil>"&&port!=""&&port!="<nil>"&&cc!=""&&cc!="<nil>" {out=append(out,ip+":"+port+"#"+strings.ToUpper(cc))}
 case string: out=append(out,parseTextNodes(v)...)
 }
 return out
}
func parseTextNodes(text string) []string {
 tokens:=strings.Fields(text);pending:=[]string{};out:=[]string{}
 pure:=regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+:\d+$`)
 for _,tok:=range tokens {
  if pure.MatchString(tok){pending=append(pending,tok);continue}
  if !strings.Contains(tok,"#"){continue}
  p:=strings.SplitN(tok,"#",2);ipport:=strings.TrimSpace(p[0]);label:=strings.TrimSpace(p[1])
  if strings.HasPrefix(ipport,"[")||!pure.MatchString(ipport){continue}
  code:=extractCountryCode(label);if code!="" {out=append(out,ipport+"#"+code)} else {pending=append(pending,ipport)}
 }
 if len(pending)>0 {
  logf("%d 个节点未能识别或缺少国家，通过可用性检测 API 查询国家...\n",len(pending))
  workers:=integer("FALLBACK_WORKERS"); if workers<1{workers=1}
  type item struct{ipp,cc string}; jobs:=make(chan string);results:=make(chan item,len(pending));var wg sync.WaitGroup
  for i:=0;i<min(workers,len(pending));i++ {wg.Add(1);go func(){defer wg.Done();for ipp:=range jobs {p:=strings.LastIndex(ipp,":");if p<0 {results<-item{ipp,""};continue};results<-item{ipp,queryCountry(ipp[:p],ipp[p+1:])}}}()}
  go func(){for _,ipp:=range pending{jobs<-ipp};close(jobs);wg.Wait();close(results)}()
  for r:=range results {if r.cc!=""{out=append(out,r.ipp+"#"+r.cc)}}
 }
 return out
}
func parseAdaptive(text string) []string {
 text=strings.TrimSpace(text);if text==""{return nil}
 if strings.HasPrefix(text,"{")||strings.HasPrefix(text,"["){var v any;if json.Unmarshal([]byte(text),&v)==nil{return parseJSONNodes(v)}}
 return parseTextNodes(text)
}
func fetchSources() []string {
 var sources []Source; raw,_:=json.Marshal(val("ADDITIONAL_SOURCES"));_ = json.Unmarshal(raw,&sources)
 nodes:=[]string{};seen:=map[string]bool{}
 for _,s:=range sources {if !s.Enabled||s.URL==""{continue}
  for attempt:=1;attempt<=integer("FETCH_MAX_RETRIES");attempt++ {
   logf("正在请求数据源 %s (尝试 %d/%d) ...\n",s.URL,attempt,integer("FETCH_MAX_RETRIES"))
   resp,b,e:=request("GET",s.URL,map[string]string{"Accept-Encoding":"gzip, deflate"},nil,duration("FETCH_CONNECT_TIMEOUT",3),duration("FETCH_TIMEOUT",3))
   if e==nil&&resp.StatusCode>=200&&resp.StatusCode<300 {
    found:=parseAdaptive(string(b));logf("从 %s 解析出 %d 个节点。\n",s.URL,len(found))
    for _,n:=range found {key:=strings.SplitN(n,"#",2)[0];if !seen[key]{seen[key]=true;nodes=append(nodes,n)}};break
   }
   if e!=nil {logf("请求失败 (%s): %v\n",s.URL,e)} else {logf("请求失败 (%s): HTTP %d\n",s.URL,resp.StatusCode)}
   if attempt<integer("FETCH_MAX_RETRIES"){sleepSec("FETCH_RETRY_DELAY")} else {logf("已尝试 %d 次，放弃该数据源。\n",integer("FETCH_MAX_RETRIES"))}
  }
 }
 logf("合并后总计 %d 个节点。\n",len(nodes));return nodes
}
func loadLines(path string) []string {b,e:=os.ReadFile(path);if e!=nil{return nil};out:=[]string{};for _,s:=range strings.Split(string(b),"\n"){s=strings.TrimSpace(s);if s!=""{out=append(out,s)}};return out}
func readCache(path string) map[string]string {out:=map[string]string{};for _,line:=range loadLines(path){p:=strings.SplitN(line,"#",2);if len(p)==2{out[strings.TrimSpace(p[0])]=strings.TrimSpace(p[1])}};return out}
func writeCache(path string, rows map[string]string) {keys:=make([]string,0,len(rows));for k:=range rows{keys=append(keys,k)};sort.Strings(keys);var b strings.Builder;for _,k:=range keys{b.WriteString(k+"#"+rows[k]+"\n")};_ = os.WriteFile(path,[]byte(b.String()),0644)}
func calibrate(nodes []string) {
 if !boolean("IP_CALIBRATION_ENABLED"){logf("IP 地区校准已禁用，跳过。\n");return}
 tokenPath:=filepath.Join(root,str("IP_CALIBRATION_TOKEN_FILE"));cachePath:=filepath.Join(root,str("IP_CALIBRATION_CACHE_FILE"));cache:=readCache(cachePath)
 for i,n:=range nodes {ipp:=strings.SplitN(n,"#",2)[0];if tag:=cache[ipp];tag!=""{nodes[i]=ipp+"#"+strings.Fields(tag)[0]}}
 tokens:=loadLines(tokenPath);if len(tokens)==0{logf("valid_tokens.txt 为空，IP 地区校准跳过。\n");return}
 logf("正在进行 token 有效性校验...\n");valid:=[]string{}
 for i,t:=range tokens {u:="https://ipinfo.io/json?token="+url.QueryEscape(t);resp,b,e:=request("GET",u,nil,nil,5*time.Second,5*time.Second);if e==nil&&resp.StatusCode==200{var d map[string]any;_ = json.Unmarshal(b,&d);if fmt.Sprint(d["country"])!=""&&fmt.Sprint(d["country"])!="<nil>"{valid=append(valid,t)}};if (i+1)%max(1,integer("IP_CALIBRATION_CONCURRENCY"))==0 {break}}
 if len(valid)==0 {logf("所有 token 均已失效，地区校准跳过。\n");notify("IP地区校准：所有token均已失效，本次校准跳过。","IP校准 Token 耗尽");return}
 logf("有效 token 数量: %d\n",len(valid))
 // Query unique uncached IPs using a bounded worker pool; token rotation avoids serially waiting on one IP.
 unique:=map[string]bool{};ipports:=map[string][]string{}
 for _,n:=range nodes {ipp:=strings.SplitN(n,"#",2)[0];if cache[ipp]!=""{continue};p:=strings.LastIndex(ipp,":");if p<0{continue};ip:=ipp[:p];unique[ip]=true;ipports[ip]=append(ipports[ip],ipp)}
 ips:=make([]string,0,len(unique));for ip:=range unique{ips=append(ips,ip)}
 if len(ips)==0{logf("所有 IP 已在缓存中，无需查询。\n")} else {
  logf("检测到 %d 个唯一 IP，开始地区查询...\n",len(ips))
  workers:=integer("IP_CALIBRATION_CONCURRENCY");if workers<1{workers=1};workers=min(workers,len(ips))
  type result struct{ip,tag string};jobs:=make(chan string);results:=make(chan result,len(ips));var wg sync.WaitGroup;var done int64
  for i:=0;i<workers;i++ {wg.Add(1);go func(){defer wg.Done();for ip:=range jobs {tag:="";for _,token:=range valid {u:="https://ipinfo.io/"+url.PathEscape(ip)+"/json?token="+url.QueryEscape(token);resp,b,e:=request("GET",u,nil,nil,5*time.Second,5*time.Second);if e==nil&&resp.StatusCode==200{var d map[string]any;_ = json.Unmarshal(b,&d);country:=fmt.Sprint(d["country"]);if country!=""&&country!="<nil>"{parts:=[]string{country};for _,k:=range []string{"region","city","org"}{if s:=strings.TrimSpace(fmt.Sprint(d[k]));s!=""&&s!="<nil>"{parts=append(parts,s)}};tag=strings.Join(parts," ");break}}};results<-result{ip,tag}}}()}
  go func(){for _,ip:=range ips{jobs<-ip};close(jobs);wg.Wait();close(results)}()
  for r:=range results {atomic.AddInt64(&done,1);if r.tag!=""{for _,ipp:=range ipports[r.ip]{cache[ipp]=r.tag}};logf("\r[%d/%d] 地区校准...",atomic.LoadInt64(&done),len(ips))}
  logf("\n")
 }
 for i,n:=range nodes {ipp:=strings.SplitN(n,"#",2)[0];if tag:=cache[ipp];tag!=""{nodes[i]=ipp+"#"+strings.Fields(tag)[0]}}
 writeCache(cachePath,cache)
}
func tcpTest(node string) (NodeResult,bool) {
 m:=nodePattern.FindStringSubmatch(node);if len(m)!=4{return NodeResult{},false}
 probes:=integer("TCP_PROBES");if probes<1{probes=1};timeout:=duration("TIMEOUT",2)
 success:=0;maxLat:=0.0
 for i:=0;i<probes;i++ {start:=time.Now();conn,e:=net.DialTimeout("tcp",net.JoinHostPort(m[1],m[2]),timeout);if e==nil {lat:=time.Since(start).Seconds();if lat>maxLat{maxLat=lat};success++;_ = conn.Close()}}
 rate:=number("MIN_SUCCESS_RATE");if success==0||float64(success)/float64(probes)<rate{return NodeResult{},false}
 return NodeResult{node,maxLat,m[3],success},true
}
func runPool[T any](items []string, workers int, fn func(string)T, progress string) []T {
 if workers<1{workers=1};workers=min(workers,max(1,len(items)))
 type pair struct{idx int; value T};jobs:=make(chan int);results:=make(chan pair,len(items));var wg sync.WaitGroup;var done int64
 for i:=0;i<workers;i++ {wg.Add(1);go func(){defer wg.Done();for idx:=range jobs {results<-pair{idx,fn(items[idx])};atomic.AddInt64(&done,1)}}()}
 go func(){for i:=range items{jobs<-i};close(jobs);wg.Wait();close(results)}()
 out:=make([]T,len(items));last:=time.Now()
 for r:=range results {out[r.idx]=r.value;if time.Since(last)>=time.Duration(max(1,integer("PROGRESS_PRINT_INTERVAL")))*time.Second||atomic.LoadInt64(&done)==int64(len(items)){logf("\r[%s] 进度：%d/%d (%.1f%%)\n",progress,atomic.LoadInt64(&done),len(items),float64(atomic.LoadInt64(&done))*100/float64(max(1,len(items))));last=time.Now()}}
 return out
}
func availability(node string) AvailabilityResult {
 m:=ipPortPattern.FindStringSubmatch(node);if len(m)!=3{return AvailabilityResult{Node:node,OK:false,Stack:"unknown"}}
 maxAttempts:=1;if boolean("AVAILABILITY_INNER_RETRY_ENABLED"){maxAttempts+=integer("AVAILABILITY_INNER_RETRY_MAX")}
 for i:=0;i<maxAttempts;i++ {u,_:=url.Parse(str("AVAILABILITY_CHECK_API"));q:=u.Query();q.Set("proxyip",m[1]+":"+m[2]);u.RawQuery=q.Encode()
  resp,b,e:=request("GET",u.String(),nil,nil,duration("AVAILABILITY_CONNECT_TIMEOUT",3),duration("AVAILABILITY_TIMEOUT",3))
  if e==nil&&resp.StatusCode==200 {var d map[string]any;_ = json.Unmarshal(b,&d);if d["success"]==true {stack:=fmt.Sprint(d["inferred_stack"]);if stack=="<nil>"{stack="unknown"};var probe map[string]any;if p,ok:=d["probe_results"].(map[string]any);ok {probe,_=p["ipv6"].(map[string]any);if probe==nil{probe,_=p["ipv4"].(map[string]any)}};exit,_:=probe["exit"].(map[string]any);return AvailabilityResult{node,true,stack,exit}}}
  if i<maxAttempts-1{sleepSec("AVAILABILITY_INNER_RETRY_DELAY")}
 }
 return AvailabilityResult{Node:node,OK:false,Stack:"unknown"}
}
func httpTest(node string) HTTPResult {
 m:=ipPortPattern.FindStringSubmatch(node);if len(m)!=3{return HTTPResult{Node:node}}
 target:="http://"+net.JoinHostPort(m[1],m[2])+"/cdn-cgi/trace";method:=strings.ToUpper(str("HTTP_TEST_METHOD"));if method!="HEAD"{method="GET"}
 n:=max(3,integer("HTTP_JITTER_SAMPLES"));lats:=[]float64{}
 for i:=0;i<n;i++ {start:=time.Now();resp,_,e:=request(method,target,map[string]string{"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"},nil,duration("HTTP_TEST_CONNECT_TIMEOUT",3),duration("HTTP_TEST_TIMEOUT",3))
  if e!=nil{return HTTPResult{Node:node,Server:"connection_error"}};lat:=float64(time.Since(start).Microseconds())/1000
  if resp.StatusCode!=400{return HTTPResult{Node:node,Server:fmt.Sprintf("status_%d",resp.StatusCode)}}
  server:=resp.Header.Get("Server");if !strings.HasPrefix(strings.ToLower(server),"cloudflare"){return HTTPResult{Node:node,Server:server}}
  lats=append(lats,lat)
 }
 var mean float64;for _,v:=range lats{mean+=v};mean/=float64(len(lats));var variance float64;for _,v:=range lats{variance+=(v-mean)*(v-mean)};variance/=float64(len(lats))
 maxlat:=0.0;for _,v:=range lats{if v>maxlat{maxlat=v}}
 return HTTPResult{Node:node,OK:true,Server:"cloudflare",Latency:maxlat,Jitter:variance}
}
func bandwidth(node string) BWResult {
 m:=ipPortPattern.FindStringSubmatch(node);if len(m)!=3{return BWResult{Node:node}}
 port,_:=strconv.Atoi(m[2]);scheme:="https";insecure:=[]string{"--insecure"}
 httpPorts:=map[int]bool{80:true,8080:true,8880:true,2052:true,2082:true,2086:true,2095:true}
 if httpPorts[port]{scheme="http";insecure=nil}
 bytesN:=int(number("BANDWIDTH_SIZE_MB")*1024*1024);template:=str("BANDWIDTH_URL_TEMPLATE")
 replacer:=strings.NewReplacer("{scheme}",scheme,"{port}",m[2],"{bytes}",strconv.Itoa(bytesN));target:=replacer.Replace(template)
 args:=[]string{"-s","-o",os.DevNull,"-w","%{size_download} %{time_starttransfer} %{time_total}","-L","-H","User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36","--http2","--noproxy","*","--resolve","speed.cloudflare.com:"+m[2]+":"+m[1],"--connect-timeout",fmt.Sprint(number("BANDWIDTH_CONNECT_TIMEOUT")),"--max-time",fmt.Sprint(number("BANDWIDTH_TIMEOUT"))}
 args=append(args,insecure...);args=append(args,target)
 ctx,cancel:=context.WithTimeout(context.Background(),duration("BANDWIDTH_TIMEOUT",3)+duration("BANDWIDTH_PROCESS_BUFFER",2));defer cancel()
 cmd:=exec.CommandContext(ctx,"curl",args...);out,e:=cmd.Output();if e!=nil{return BWResult{Node:node}}
 p:=strings.Fields(string(out));if len(p)<3{return BWResult{Node:node}}
 size,_:=strconv.ParseFloat(p[0],64);total,_:=strconv.ParseFloat(p[2],64)
 if size<float64(bytesN)||total<=0{return BWResult{Node:node}}
 return BWResult{Node:node,Speed:size*8/(total*1e6)}
}
func writeIP(nodes []string,speed,lat,httpLat,jitter map[string]float64) error {
 var b strings.Builder
 if boolean("AD_HEADER_ENABLED"){for _,s:=range stringsVal("AD_HEADER_LINES"){b.WriteString(s+"\n")}}
 for _,n:=range nodes {line:=n;if boolean("IP_TXT_SHOW_BANDWIDTH"){if v,ok:=speed[n];ok{line+=fmt.Sprintf(" %.2f Mbps",v)}};if boolean("IP_TXT_SHOW_HTTP_LATENCY"){if v,ok:=httpLat[n];ok{line+=fmt.Sprintf(" %.2f ms",v)}};if boolean("IP_TXT_SHOW_HTTP_JITTER"){if v,ok:=jitter[n];ok{line+=fmt.Sprintf(" %.2f ms",v)}};if boolean("IP_TXT_SHOW_LATENCY"){if v,ok:=lat[n];ok{line+=fmt.Sprintf(" %.2f ms",v*1000)}};if boolean("AD_PERLINE_ENABLED"){line+=str("AD_PERLINE_TEXT")};b.WriteString(line+"\n")}
 if boolean("AD_FOOTER_ENABLED"){for _,s:=range stringsVal("AD_FOOTER_LINES"){b.WriteString(s+"\n")}}
 return os.WriteFile(filepath.Join(root,str("OUTPUT_FILE")),[]byte(b.String()),0644)
}
func syncGitHub() {
 script:="git_sync.sh";args:=[]string{filepath.Join(root,script)}
 if os.PathSeparator=='\\' {script="git_sync.ps1";args=[]string{"powershell","-NoProfile","-NonInteractive","-ExecutionPolicy","Bypass","-File",filepath.Join(root,script)}}
 if _,e:=os.Stat(filepath.Join(root,script));e!=nil {logf("未找到 %s，跳过 GitHub 同步。\n",script);return}
 for i:=1;i<=integer("GITHUB_SYNC_MAX_RETRIES");i++ {logf("\n正在同步到 GitHub (尝试 %d/%d)...\n",i,integer("GITHUB_SYNC_MAX_RETRIES"));ctx,cancel:=context.WithTimeout(context.Background(),duration("GIT_SYNC_PROCESS_TIMEOUT",180));cmd:=exec.CommandContext(ctx,args[0],args[1:]...);cmd.Dir=root;out,e:=cmd.CombinedOutput();cancel();if e==nil{logf("已自动推送到 GitHub。\n");return};logf("推送失败: %v\n%s\n",e,string(out));if i<integer("GITHUB_SYNC_MAX_RETRIES"){sleepSec("GITHUB_SYNC_RETRY_DELAY")}}
 notify("GitHub 推送失败，请检查网络或仓库状态。","GitHub 推送失败");logf("已尝试多次推送，均失败。\n")
}
func updateDNS(nodes []string, bw []BWResult, av map[string]AvailabilityResult, lat,httpLat,jitter,speed map[string]float64) {
 if !boolean("CF_ENABLED"){logf("Cloudflare DNS 批量更新未启用。\n");return}
 recordType:=strings.ToUpper(str("DNS_RECORD_TYPE"));if recordType!="A"&&recordType!="TXT"{logf("不支持的 DNS_RECORD_TYPE: %s，已跳过 DNS 更新。\n",recordType);return}
 content:=[]string{};seen:=map[string]bool{};targetCount:=integer("DNS_UPDATE_TARGET_COUNT")
 for _,r:=range bw {m:=ipPortPattern.FindStringSubmatch(r.Node);if len(m)!=3{continue};if recordType=="A"&&m[2]!="443"{continue}
  if boolean("FILTER_IPV6_AVAILABILITY"){if a,ok:=av[r.Node];ok&&a.Stack=="ipv6_only"{continue}}
  if boolean("FILTER_BLOCKED_COUNTRIES_ENABLED"){country:=strings.SplitN(strings.SplitN(r.Node,"#",2)[1]," ",2)[0];blocked:=false;for _,c:=range stringsVal("BLOCKED_COUNTRIES"){if strings.EqualFold(c,country){blocked=true;break}};if blocked{continue}}
  value:=m[1];if recordType=="TXT"{value=m[1]+":"+m[2]};if !seen[value]{seen[value]=true;content=append(content,value)};if len(content)>=targetCount{break}
 }
 if len(content)==0&&recordType=="A"{for _,n:=range nodes{m:=ipPortPattern.FindStringSubmatch(n);if len(m)==3&&!seen[m[1]]{seen[m[1]]=true;content=append(content,m[1]);if len(content)>=targetCount{break}}}}
 if len(content)==0 {logf("没有可用的 IP 用于 DNS 更新，跳过。\n");notify("没有可用的 IP 用于 DNS 更新，跳过。","DNS 更新跳过");return}
 logf("准备更新 Cloudflare DNS：%d 条记录，类型 %s\n",len(content),recordType)
 headers:=map[string]string{"Authorization":"Bearer "+str("CF_API_TOKEN"),"Content-Type":"application/json"}
 for attempt:=1;attempt<=integer("DNS_UPDATE_MAX_RETRIES");attempt++ {
  base:="https://api.cloudflare.com/client/v4/zones/"+url.PathEscape(str("CF_ZONE_ID"))+"/dns_records"
  u:=base+"?type="+recordType+"&name="+url.QueryEscape(str("CF_DNS_RECORD_NAME"))
  resp,b,e:=request("GET",u,headers,nil,duration("CF_DNS_CONNECT_TIMEOUT",3),duration("CF_DNS_READ_TIMEOUT",3))
  if e==nil&&resp.StatusCode>=200&&resp.StatusCode<300 {
   var data map[string]any;_ = json.Unmarshal(b,&data);success,_:=data["success"].(bool)
   if success {
    recs,_:=data["result"].([]any);deletes:=[]any{};for _,x:=range recs{if r,ok:=x.(map[string]any);ok{if id,ok:=r["id"];ok{deletes=append(deletes,map[string]any{"id":id})}}}
    posts:=[]any{};for _,v:=range content{r:=map[string]any{"name":str("CF_DNS_RECORD_NAME"),"type":recordType,"content":v,"ttl":integer("CF_TTL")};if recordType=="A"{r["proxied"]=boolean("CF_PROXIED")};posts=append(posts,r)}
    body,_:=json.Marshal(map[string]any{"deletes":deletes,"posts":posts})
    rr,bb,ee:=request("POST",base+"/batch",headers,body,duration("CF_DNS_CONNECT_TIMEOUT",3),duration("CF_DNS_READ_TIMEOUT",3))
    var res map[string]any;if ee==nil{_ = json.Unmarshal(bb,&res)}
    if ee==nil&&rr.StatusCode>=200&&rr.StatusCode<300&&res["success"]==true{logf("Cloudflare DNS 批量更新成功！共 %d 条记录。\n",len(content));return}
    if ee!=nil{e=ee}else{e=fmt.Errorf("批量更新失败: HTTP %d",rr.StatusCode)}
   } else {e=errors.New("查询 DNS 记录失败")}
  } else if e==nil {e=fmt.Errorf("HTTP %d",resp.StatusCode)}
  logf("[尝试 %d/%d] DNS 更新出错: %v\n",attempt,integer("DNS_UPDATE_MAX_RETRIES"),e);if attempt<integer("DNS_UPDATE_MAX_RETRIES"){sleepSec("DNS_UPDATE_RETRY_DELAY")}
 }
 notify("Cloudflare DNS 更新失败，请检查配置和网络。","DNS 更新失败")
}
func main() {
 root,_=filepath.Abs(filepath.Dir(os.Args[0]))
 // go run 时从当前工作目录读取配置，二进制运行时从二进制所在目录读取。
 if _,e:=os.Stat(filepath.Join(root,"config.json"));e!=nil {root,_=os.Getwd()}
 if e:=loadConfig();e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)}
 if boolean("FORCE_DIRECT"){for _,k:=range []string{"HTTP_PROXY","HTTPS_PROXY","http_proxy","https_proxy","ALL_PROXY","all_proxy"}{_ = os.Unsetenv(k)};_ = os.Setenv("NO_PROXY","*")}
 lockPath:=filepath.Join(root,".run.lock");lock,e:=os.OpenFile(lockPath,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600);if e!=nil{logf("检测到本程序已在运行或锁文件残留，本次启动自动退出。\n");return};_,_=lock.WriteString(strconv.Itoa(os.Getpid()));lock.Close();defer os.Remove(lockPath)
 mode:="全局最优"+strconv.Itoa(integer("GLOBAL_TOP_N"))+"个";if !boolean("USE_GLOBAL_MODE"){mode="每个国家最优"+strconv.Itoa(integer("PER_COUNTRY_TOP_N"))+"个"}
 logf("当前模式：%s，每个节点测试 %d 次 TCP 连接\n",mode,integer("TCP_PROBES"));logf("最低成功率要求：%.0f%%\n",number("MIN_SUCCESS_RATE")*100)
 nodes:=fetchSources();if len(nodes)==0{logf("没有获取到任何有效节点，退出。\n");return}
 calibrate(nodes)
 if boolean("PRE_FILTER_PORT_ENABLED"){allowed:=map[string]bool{};for _,v:=range stringsVal("PRE_FILTER_PORTS"){allowed[fmt.Sprint(v)]=true};before:=len(nodes);out:=[]string{};for _,n:=range nodes{m:=ipPortPattern.FindStringSubmatch(n);if len(m)==3&&allowed[m[2]]{out=append(out,n)}};nodes=out;logf("前置端口过滤：%d -> %d 个节点\n",before,len(nodes))}
 if boolean("PRE_FILTER_BLOCKED_ENABLED"){blocked:=map[string]bool{};for _,v:=range stringsVal("PRE_FILTER_BLOCKED_COUNTRIES"){blocked[strings.ToUpper(v)]=true};before:=len(nodes);out:=[]string{};for _,n:=range nodes{p:=strings.SplitN(n,"#",2);if len(p)==2&&!blocked[strings.ToUpper(strings.Fields(p[1])[0])]{out=append(out,n)}};nodes=out;logf("前置黑名单过滤：%d -> %d 个节点\n",before,len(nodes))}
 if boolean("FILTER_COUNTRIES_ENABLED"){allow:=map[string]bool{};for _,c:=range stringsVal("ALLOWED_COUNTRIES"){allow[strings.ToUpper(c)]=true};out:=[]string{};for _,n:=range nodes{p:=strings.SplitN(n,"#",2);if len(p)==2&&allow[strings.ToUpper(strings.Fields(p[1])[0])]{out=append(out,n)}};nodes=out;logf("国家白名单过滤后剩余 %d 个节点\n",len(nodes))}
 if len(nodes)==0{logf("过滤后无任何节点，退出。\n");return}
 logf("开始 TCP 连接测试（超时 %gs，并发 %d）...\n",number("TIMEOUT"),integer("MAX_WORKERS"))
 tcp:=runPool(nodes,integer("MAX_WORKERS"),func(n string)NodeResult{r,_:=tcpTest(n);return r},"TCP")
 results:=[]NodeResult{};for _,r:=range tcp{if r.Node!=""{results=append(results,r)}}
 if len(results)==0{logf("没有通过成功率筛选的节点，请检查网络或降低 MIN_SUCCESS_RATE。\n");return}
 sort.SliceStable(results,func(i,j int)bool{if results[i].Success!=results[j].Success{return results[i].Success>results[j].Success};return results[i].Latency<results[j].Latency})
 latency:=map[string]float64{};for _,r:=range results{latency[r.Node]=r.Latency}
 candidates:=[]string{};countryNodes:=map[string][]NodeResult{}
 if boolean("USE_GLOBAL_MODE"){for _,r:=range results{if len(candidates)>=integer("BANDWIDTH_CANDIDATES"){break};candidates=append(candidates,r.Node)}} else {for _,r:=range results{countryNodes[r.Country]=append(countryNodes[r.Country],r)};base:=max(1,integer("BANDWIDTH_CANDIDATES")/max(1,len(countryNodes)));for _,rs:=range countryNodes{sort.SliceStable(rs,func(i,j int)bool{if rs[i].Success!=rs[j].Success{return rs[i].Success>rs[j].Success};return rs[i].Latency<rs[j].Latency});for i:=0;i<min(base,len(rs));i++{candidates=append(candidates,rs[i].Node)}}}
 logf("TCP 测试通过 %d 个节点，%d 个节点进入候选池。\n",len(results),len(candidates));if len(candidates)==0{return}
 avMap:=map[string]AvailabilityResult{};afterAv:=candidates
 if boolean("TEST_AVAILABILITY"){passed:=[]string{};for attempt:=1;attempt<=integer("AVAILABILITY_RETRY_MAX");attempt++ {logf("[可用性检测] 第 %d 轮检测...\n",attempt);arr:=runPool(afterAv,integer("AVAILABILITY_WORKERS"),func(n string)AvailabilityResult{return availability(n)},"可用性检测");passed=passed[:0];for _,r:=range arr{avMap[r.Node]=r;if r.OK{passed=append(passed,r.Node)}};if len(passed)>0{afterAv=passed;break};if attempt<integer("AVAILABILITY_RETRY_MAX"){sleepSec("AVAILABILITY_RETRY_DELAY")}}
  if len(passed)==0{logf("可用性检测全部失败，降级使用原候选列表。\n");notify("IP 可用性检测全部失败，已跳过过滤。","可用性检测全部失败")}else{afterAv=passed}}
 httpLat:=map[string]float64{};jitter:=map[string]float64{};afterHTTP:=afterAv
 if boolean("HTTP_TEST_ENABLED"){var passed []string;for round:=1;round<=integer("HTTP_TEST_MAX_ROUNDS");round++ {logf("[HTTP检测] 第 %d 轮检测...\n",round);arr:=runPool(afterAv,integer("HTTP_TEST_WORKERS"),func(n string)HTTPResult{return httpTest(n)},"HTTP检测");passed=nil;for _,r:=range arr{if r.OK{passed=append(passed,r.Node);httpLat[r.Node]=r.Latency;jitter[r.Node]=r.Jitter}};if len(passed)>0{afterHTTP=passed;break};if round<integer("HTTP_TEST_MAX_ROUNDS"){sleepSec("HTTP_TEST_ROUND_DELAY")}}
  if len(passed)==0{afterHTTP=afterAv;httpLat=map[string]float64{};jitter=map[string]float64{};notify("HTTP检测全部失败，已降级使用过滤前候选列表。","HTTP检测全部失败")} }
 if _,e:=exec.LookPath("curl");e!=nil{logf("未检测到 curl 命令，带宽测速将跳过。\n")}
 bw:=[]BWResult{}
 for attempt:=1;attempt<=integer("BANDWIDTH_RETRY_MAX");attempt++ {logf("[带宽测速] 第 %d 轮测试...\n",attempt);arr:=runPool(afterHTTP,integer("BANDWIDTH_WORKERS"),bandwidth,"带宽测速");bw=nil;for _,r:=range arr{if r.Speed>0{bw=append(bw,r)}};sort.SliceStable(bw,func(i,j int)bool{return bw[i].Speed>bw[j].Speed});if len(bw)>0{break};if attempt<integer("BANDWIDTH_RETRY_MAX"){sleepSec("BANDWIDTH_RETRY_DELAY")}}
 speed:=map[string]float64{};selected:=[]string{}
 if len(bw)==0 {logf("带宽测速多次重试仍无有效结果，将使用 TCP 筛选结果作为最终节点。\n");notify("带宽测速全部失败，已降级使用 TCP 排序节点。","带宽测速全部失败");if boolean("USE_GLOBAL_MODE"){for _,r:=range results{if len(selected)>=integer("GLOBAL_TOP_N"){break};selected=append(selected,r.Node)}}else{for _,rs:=range countryNodes{sort.SliceStable(rs,func(i,j int)bool{if rs[i].Success!=rs[j].Success{return rs[i].Success>rs[j].Success};return rs[i].Latency<rs[j].Latency});for i:=0;i<min(integer("PER_COUNTRY_TOP_N"),len(rs));i++{selected=append(selected,rs[i].Node)}}}
 } else {
  for _,r:=range bw{speed[r.Node]=r.Speed}
  score:=map[string]float64{};for _,r:=range bw{tl:=latency[r.Node];hl:=httpLat[r.Node]/1000;j:=jitter[r.Node]/1000;penalty:=1+number("TCP_LATENCY_WEIGHT")*tl+number("HTTP_LATENCY_WEIGHT")*hl+number("JITTER_WEIGHT")*j;if penalty<=0{penalty=1};score[r.Node]=number("SPEED_WEIGHT")*r.Speed/penalty}
  sort.SliceStable(bw,func(i,j int)bool{return score[bw[i].Node]>score[bw[j].Node]})
  if boolean("USE_GLOBAL_MODE"){for i:=0;i<min(integer("GLOBAL_TOP_N"),len(bw));i++{selected=append(selected,bw[i].Node)}} else {count:=map[string]int{};for _,r:=range bw{country:="";p:=strings.SplitN(r.Node,"#",2);if len(p)==2{country=strings.Fields(p[1])[0]};if country!=""&&count[country]<integer("PER_COUNTRY_TOP_N"){selected=append(selected,r.Node);count[country]++}}}
 }
 logf("\n================ 最终优选节点 ================\n");for i,n:=range selected{line:=fmt.Sprintf("%d. %s",i+1,n);if v,ok:=speed[n];ok{line+=fmt.Sprintf(" 速度 %.2f Mbps",v)};if v,ok:=httpLat[n];ok{line+=fmt.Sprintf(" HTTP延迟 %.2f ms",v)};if v,ok:=jitter[n];ok{line+=fmt.Sprintf(" 抖动 %.2f ms",v)};if v,ok:=latency[n];ok{line+=fmt.Sprintf(" TCP延迟 %.2f ms",v*1000)};logf("%s\n",line)}
 if e:=writeIP(selected,speed,latency,httpLat,jitter);e!=nil{logf("写入结果失败: %v\n",e);return};logf("结果已保存到 %s（共 %d 个节点）\n",str("OUTPUT_FILE"),len(selected))
 ipList:=[]string{};for _,n:=range selected{p:=strings.SplitN(n,":",2);ipList=append(ipList,p[0])}
 updateDNS(selected,bw,avMap,latency,httpLat,jitter,speed)
 syncGitHub()
}
