// Cloudflare IP 优选工具 - Go 迁移版
// 只使用 Go 标准库；读取原有 config.json，保留原参数名/值。
// 带宽测速继续调用原配置所用的 curl；GitHub 同步继续使用原 git_sync.sh / git_sync.ps1。
package main

import (
 "bytes"
 "context"
 "crypto/tls"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "math"
 "net"
 "net/http"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
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
"阿富汗": "AF",
"奥兰群岛": "AX",
"阿尔巴尼亚": "AL",
"阿尔及利亚": "DZ",
"美属萨摩亚": "AS",
"安道尔": "AD",
"安哥拉": "AO",
"安圭拉": "AI",
"南极洲": "AQ",
"安提瓜和巴布达": "AG",
"阿根廷": "AR",
"亚美尼亚": "AM",
"阿鲁巴": "AW",
"澳大利亚": "AU",
"奥地利": "AT",
"阿塞拜疆": "AZ",
"巴哈马": "BS",
"巴林": "BH",
"孟加拉国": "BD",
"孟加拉": "BD",
"巴巴多斯": "BB",
"白俄罗斯": "BY",
"比利时": "BE",
"伯利兹": "BZ",
"贝宁": "BJ",
"百慕大": "BM",
"不丹": "BT",
"玻利维亚": "BO",
"波黑": "BA",
"波斯尼亚和黑塞哥维那": "BA",
"博茨瓦纳": "BW",
"布维岛": "BV",
"巴西": "BR",
"英属印度洋领地": "IO",
"文莱": "BN",
"保加利亚": "BG",
"布基纳法索": "BF",
"布隆迪": "BI",
"柬埔寨": "KH",
"喀麦隆": "CM",
"加拿大": "CA",
"佛得角": "CV",
"开曼群岛": "KY",
"中非": "CF",
"乍得": "TD",
"智利": "CL",
"中国": "CN",
"圣诞岛": "CX",
"科科斯(基林)群岛": "CC",
"哥伦比亚": "CO",
"科摩罗": "KM",
"刚果(布)": "CG",
"刚果（布）": "CG",
"刚果(金)": "CD",
"刚果（金）": "CD",
"库克群岛": "CK",
"哥斯达黎加": "CR",
"科特迪瓦": "CI",
"克罗地亚": "HR",
"古巴": "CU",
"塞浦路斯": "CY",
"捷克": "CZ",
"丹麦": "DK",
"吉布提": "DJ",
"多米尼克": "DM",
"多米尼加": "DO",
"厄瓜多尔": "EC",
"埃及": "EG",
"萨尔瓦多": "SV",
"赤道几内亚": "GQ",
"厄立特里亚": "ER",
"爱沙尼亚": "EE",
"埃塞俄比亚": "ET",
"福克兰群岛(马尔维纳斯)": "FK",
"法罗群岛": "FO",
"斐济": "FJ",
"芬兰": "FI",
"法国": "FR",
"法属圭亚那": "GF",
"法属波利尼西亚": "PF",
"法属南部领地": "TF",
"加蓬": "GA",
"冈比亚": "GM",
"格鲁吉亚": "GE",
"德国": "DE",
"加纳": "GH",
"直布罗陀": "GI",
"希腊": "GR",
"格陵兰": "GL",
"格林纳达": "GD",
"瓜德罗普": "GP",
"关岛": "GU",
"危地马拉": "GT",
"根西岛": "GG",
"几内亚": "GN",
"几内亚比绍": "GW",
"圭亚那": "GY",
"海地": "HT",
"赫德岛和麦克唐纳群岛": "HM",
"梵蒂冈": "VA",
"洪都拉斯": "HN",
"香港": "HK",
"匈牙利": "HU",
"冰岛": "IS",
"印度": "IN",
"印度尼西亚": "ID",
"伊朗": "IR",
"伊拉克": "IQ",
"爱尔兰": "IE",
"马恩岛": "IM",
"以色列": "IL",
"意大利": "IT",
"牙买加": "JM",
"日本": "JP",
"泽西岛": "JE",
"约旦": "JO",
"哈萨克斯坦": "KZ",
"肯尼亚": "KE",
"基里巴斯": "KI",
"朝鲜": "KP",
"韩国": "KR",
"科威特": "KW",
"吉尔吉斯斯坦": "KG",
"老挝": "LA",
"拉脱维亚": "LV",
"黎巴嫩": "LB",
"莱索托": "LS",
"利比里亚": "LR",
"利比亚": "LY",
"列支敦士登": "LI",
"立陶宛": "LT",
"卢森堡": "LU",
"澳门": "MO",
"北马其顿": "MK",
"马其顿": "MK",
"马达加斯加": "MG",
"马拉维": "MW",
"马来西亚": "MY",
"马尔代夫": "MV",
"马里": "ML",
"马耳他": "MT",
"马绍尔群岛": "MH",
"马提尼克": "MQ",
"毛里塔尼亚": "MR",
"毛里求斯": "MU",
"马约特": "YT",
"墨西哥": "MX",
"密克罗尼西亚": "FM",
"摩尔多瓦": "MD",
"摩纳哥": "MC",
"蒙古": "MN",
"黑山": "ME",
"蒙特塞拉特": "MS",
"摩洛哥": "MA",
"莫桑比克": "MZ",
"缅甸": "MM",
"纳米比亚": "NA",
"瑙鲁": "NR",
"尼泊尔": "NP",
"荷兰": "NL",
"新喀里多尼亚": "NC",
"新西兰": "NZ",
"尼加拉瓜": "NI",
"尼日尔": "NE",
"尼日利亚": "NG",
"纽埃": "NU",
"诺福克岛": "NF",
"北马里亚纳群岛": "MP",
"挪威": "NO",
"阿曼": "OM",
"巴基斯坦": "PK",
"帕劳": "PW",
"巴勒斯坦": "PS",
"巴拿马": "PA",
"巴布亚新几内亚": "PG",
"巴拉圭": "PY",
"秘鲁": "PE",
"菲律宾": "PH",
"皮特凯恩": "PN",
"波兰": "PL",
"葡萄牙": "PT",
"波多黎各": "PR",
"卡塔尔": "QA",
"留尼汪": "RE",
"罗马尼亚": "RO",
"俄罗斯": "RU",
"卢旺达": "RW",
"圣巴泰勒米": "BL",
"圣赫勒拿": "SH",
"圣基茨和尼维斯": "KN",
"圣卢西亚": "LC",
"圣马丁": "MF",
"圣皮埃尔和密克隆": "PM",
"圣文森特和格林纳丁斯": "VC",
"萨摩亚": "WS",
"圣马力诺": "SM",
"圣多美和普林西比": "ST",
"沙特阿拉伯": "SA",
"沙特": "SA",
"塞内加尔": "SN",
"塞尔维亚": "RS",
"塞舌尔": "SC",
"塞拉利昂": "SL",
"新加坡": "SG",
"圣马丁(荷兰)": "SX",
"斯洛伐克": "SK",
"斯洛文尼亚": "SI",
"所罗门群岛": "SB",
"索马里": "SO",
"南非": "ZA",
"南乔治亚和南桑威奇群岛": "GS",
"南苏丹": "SS",
"西班牙": "ES",
"斯里兰卡": "LK",
"苏丹": "SD",
"苏里南": "SR",
"斯瓦尔巴和扬马延": "SJ",
"斯威士兰": "SZ",
"瑞典": "SE",
"瑞士": "CH",
"叙利亚": "SY",
"台湾": "TW",
"塔吉克斯坦": "TJ",
"坦桑尼亚": "TZ",
"泰国": "TH",
"东帝汶": "TL",
"多哥": "TG",
"托克劳": "TK",
"汤加": "TO",
"特立尼达和多巴哥": "TT",
"突尼斯": "TN",
"土耳其": "TR",
"土库曼斯坦": "TM",
"特克斯和凯科斯群岛": "TC",
"图瓦卢": "TV",
"乌干达": "UG",
"乌克兰": "UA",
"阿联酋": "AE",
"英国": "GB",
"美国": "US",
"美国本土外小岛屿": "UM",
"乌拉圭": "UY",
"乌兹别克斯坦": "UZ",
"瓦努阿图": "VU",
"委内瑞拉": "VE",
"越南": "VN",
"英属维尔京群岛": "VG",
"美属维尔京群岛": "VI",
"瓦利斯和富图纳": "WF",
"西撒哈拉": "EH",
"也门": "YE",
"赞比亚": "ZM",
"津巴布韦": "ZW",
}
var alpha3ToAlpha2 = map[string]string{
"AFG": "AF",
"ALA": "AX",
"ALB": "AL",
"DZA": "DZ",
"ASM": "AS",
"AND": "AD",
"AGO": "AO",
"AIA": "AI",
"ATA": "AQ",
"ATG": "AG",
"ARG": "AR",
"ARM": "AM",
"ABW": "AW",
"AUS": "AU",
"AUT": "AT",
"AZE": "AZ",
"BHS": "BS",
"BHR": "BH",
"BGD": "BD",
"BRB": "BB",
"BLR": "BY",
"BEL": "BE",
"BLZ": "BZ",
"BEN": "BJ",
"BMU": "BM",
"BTN": "BT",
"BOL": "BO",
"BIH": "BA",
"BWA": "BW",
"BVT": "BV",
"BRA": "BR",
"IOT": "IO",
"BRN": "BN",
"BGR": "BG",
"BFA": "BF",
"BDI": "BI",
"KHM": "KH",
"CMR": "CM",
"CAN": "CA",
"CPV": "CV",
"CYM": "KY",
"CAF": "CF",
"TCD": "TD",
"CHL": "CL",
"CHN": "CN",
"CXR": "CX",
"CCK": "CC",
"COL": "CO",
"COM": "KM",
"COG": "CG",
"COD": "CD",
"COK": "CK",
"CRI": "CR",
"CIV": "CI",
"HRV": "HR",
"CUB": "CU",
"CYP": "CY",
"CZE": "CZ",
"DNK": "DK",
"DJI": "DJ",
"DMA": "DM",
"DOM": "DO",
"ECU": "EC",
"EGY": "EG",
"SLV": "SV",
"GNQ": "GQ",
"ERI": "ER",
"EST": "EE",
"ETH": "ET",
"FLK": "FK",
"FRO": "FO",
"FJI": "FJ",
"FIN": "FI",
"FRA": "FR",
"GUF": "GF",
"PYF": "PF",
"ATF": "TF",
"GAB": "GA",
"GMB": "GM",
"GEO": "GE",
"DEU": "DE",
"GHA": "GH",
"GIB": "GI",
"GRC": "GR",
"GRL": "GL",
"GRD": "GD",
"GLP": "GP",
"GUM": "GU",
"GTM": "GT",
"GGY": "GG",
"GIN": "GN",
"GNB": "GW",
"GUY": "GY",
"HTI": "HT",
"HMD": "HM",
"VAT": "VA",
"HND": "HN",
"HKG": "HK",
"HUN": "HU",
"ISL": "IS",
"IND": "IN",
"IDN": "ID",
"IRN": "IR",
"IRQ": "IQ",
"IRL": "IE",
"IMN": "IM",
"ISR": "IL",
"ITA": "IT",
"JAM": "JM",
"JPN": "JP",
"JEY": "JE",
"JOR": "JO",
"KAZ": "KZ",
"KEN": "KE",
"KIR": "KI",
"PRK": "KP",
"KOR": "KR",
"KWT": "KW",
"KGZ": "KG",
"LAO": "LA",
"LVA": "LV",
"LBN": "LB",
"LSO": "LS",
"LBR": "LR",
"LBY": "LY",
"LIE": "LI",
"LTU": "LT",
"LUX": "LU",
"MAC": "MO",
"MKD": "MK",
"MDG": "MG",
"MWI": "MW",
"MYS": "MY",
"MDV": "MV",
"MLI": "ML",
"MLT": "MT",
"MHL": "MH",
"MTQ": "MQ",
"MRT": "MR",
"MUS": "MU",
"MYT": "YT",
"MEX": "MX",
"FSM": "FM",
"MDA": "MD",
"MCO": "MC",
"MNG": "MN",
"MNE": "ME",
"MSR": "MS",
"MAR": "MA",
"MOZ": "MZ",
"MMR": "MM",
"NAM": "NA",
"NRU": "NR",
"NPL": "NP",
"NLD": "NL",
"NCL": "NC",
"NZL": "NZ",
"NIC": "NI",
"NER": "NE",
"NGA": "NG",
"NIU": "NU",
"NFK": "NF",
"MNP": "MP",
"NOR": "NO",
"OMN": "OM",
"PAK": "PK",
"PLW": "PW",
"PSE": "PS",
"PAN": "PA",
"PNG": "PG",
"PRY": "PY",
"PER": "PE",
"PHL": "PH",
"PCN": "PN",
"POL": "PL",
"PRT": "PT",
"PRI": "PR",
"QAT": "QA",
"REU": "RE",
"ROU": "RO",
"RUS": "RU",
"RWA": "RW",
"BLM": "BL",
"SHN": "SH",
"KNA": "KN",
"LCA": "LC",
"MAF": "MF",
"SPM": "PM",
"VCT": "VC",
"WSM": "WS",
"SMR": "SM",
"STP": "ST",
"SAU": "SA",
"SEN": "SN",
"SRB": "RS",
"SYC": "SC",
"SLE": "SL",
"SGP": "SG",
"SXM": "SX",
"SVK": "SK",
"SVN": "SI",
"SLB": "SB",
"SOM": "SO",
"ZAF": "ZA",
"SGS": "GS",
"SSD": "SS",
"ESP": "ES",
"LKA": "LK",
"SDN": "SD",
"SUR": "SR",
"SJM": "SJ",
"SWZ": "SZ",
"SWE": "SE",
"CHE": "CH",
"SYR": "SY",
"TWN": "TW",
"TJK": "TJ",
"TZA": "TZ",
"THA": "TH",
"TLS": "TL",
"TGO": "TG",
"TKL": "TK",
"TON": "TO",
"TTO": "TT",
"TUN": "TN",
"TUR": "TR",
"TKM": "TM",
"TCA": "TC",
"TUV": "TV",
"UGA": "UG",
"UKR": "UA",
"ARE": "AE",
"GBR": "GB",
"USA": "US",
"UMI": "UM",
"URY": "UY",
"UZB": "UZ",
"VUT": "VU",
"VEN": "VE",
"VNM": "VN",
"VGB": "VG",
"VIR": "VI",
"WLF": "WF",
"ESH": "EH",
"YEM": "YE",
"ZMB": "ZM",
"ZWE": "ZW",
}
var codeSet = map[string]bool{}
var nodePattern = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#(.+)$`)
var ipPortPattern = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):(\d+)#`)
var muLog sync.Mutex

func init() {
 for _, code := range cnToCode { codeSet[code] = true }
}

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
 b,e:=os.ReadFile(p)
 if e!=nil {
  if errors.Is(e, os.ErrNotExist) {
   logf("未找到配置文件 %s，将使用内置默认配置运行。\n", p)
   logf("你可根据需要创建 config.json 文件（参考文档），程序会自动识别。\n")
   cfg = Config{}
  } else { return fmt.Errorf("读取 config.json 失败: %w", e) }
 } else if e=json.Unmarshal(b,&cfg);e!=nil { return fmt.Errorf("config.json 格式错误: %w",e) }
 // 保留原配置中的所有键和所有值；只为缺少的键补上与原脚本相同的默认值。
 defaults:=map[string]any{
 "USE_GLOBAL_MODE":true,"GLOBAL_TOP_N":15,"PER_COUNTRY_TOP_N":1,"BANDWIDTH_CANDIDATES":150,"TCP_PROBES":1,"MIN_SUCCESS_RATE":1.0,"TCP_LATENCY_WEIGHT":0.0,"TIMEOUT":2.0,"SOCKET_DEFAULT_TIMEOUT":3,"PROGRESS_PRINT_INTERVAL":1,
 "FILTER_COUNTRIES_ENABLED":false,"ALLOWED_COUNTRIES":[]any{"US"},"PRE_FILTER_BLOCKED_ENABLED":true,"PRE_FILTER_BLOCKED_COUNTRIES":[]any{"CN"},"PRE_FILTER_PORT_ENABLED":true,"PRE_FILTER_PORTS":[]any{443},
 "ENABLE_WXPUSHER":true,"WXPUSHER_APP_TOKEN":"your_app_token_here","WXPUSHER_UIDS":[]any{"your_uid_here"},"WXPUSHER_API_URL":"https://wxpusher.zjiecode.com/api/send/message","NOTIFY_TIMEOUT":3,"NOTIFY_CONNECT_TIMEOUT":3,
 "CF_ENABLED":true,"CF_API_TOKEN":"your_CF_API_TOKEN","CF_ZONE_ID":"your_CF_ZONE_ID","CF_DNS_RECORD_NAME":"your_CF_DNS_RECORD_NAME","CF_TTL":60,"CF_PROXIED":false,"CF_DNS_CONNECT_TIMEOUT":3,"CF_DNS_READ_TIMEOUT":3,"DNS_RECORD_TYPE":"TXT",
 "ADDITIONAL_SOURCES":[]any{map[string]any{"url":"https://zip.cm.edu.kg/all.txt","enabled":true},map[string]any{"url":"https://countrymerge.pages.dev/all.txt","enabled":true}},"FETCH_MAX_RETRIES":3,"FETCH_RETRY_DELAY":3,"FETCH_TIMEOUT":3,"FETCH_CONNECT_TIMEOUT":3,"IP_CALIBRATION_ENABLED":false,"TOKEN_FAILURE_THRESHOLD":3,"IP_CALIBRATION_MIN_INTERVAL":0.1,"IP_CALIBRATION_TOKEN_FILE":"valid_tokens.txt","IP_CALIBRATION_CACHE_FILE":"ipinfo_cache.txt","OUTPUT_FILE":"ip.txt","ENABLE_LOGGING":false,"LOG_FILE":"cfnb.log","FORCE_DIRECT":false,
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
var httpClientMu sync.Mutex
var httpClients = map[string]*http.Client{}

func newHTTPClient(connect, read time.Duration) *http.Client {
 key := connect.String() + "/" + read.String() + "/" + strconv.FormatBool(boolean("FORCE_DIRECT"))
 httpClientMu.Lock()
 defer httpClientMu.Unlock()
 if client := httpClients[key]; client != nil {
  return client
 }
 tr := &http.Transport{
  DialContext: (&net.Dialer{Timeout: connect, KeepAlive: 30 * time.Second}).DialContext,
  TLSHandshakeTimeout: connect,
  ResponseHeaderTimeout: read,
  IdleConnTimeout: 30 * time.Second,
  MaxIdleConns: 128,
  MaxIdleConnsPerHost: 64,
  TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
 }
 if !boolean("FORCE_DIRECT") {
  tr.Proxy = http.ProxyFromEnvironment
 }
 client := &http.Client{Transport: tr, Timeout: connect + read + 2*time.Second}
 httpClients[key] = client
 return client
}
func request(method, target string, headers map[string]string, body []byte, connect, read time.Duration) (*http.Response,[]byte,error) {
 req,e:=http.NewRequest(method,target,bytes.NewReader(body));if e!=nil{return nil,nil,e}
 for k,v:=range headers {req.Header.Set(k,v)}
 resp,e:=newHTTPClient(connect,read).Do(req);if e!=nil{return nil,nil,e}
 b,re:=io.ReadAll(resp.Body);resp.Body.Close();if re!=nil{return resp,b,re};return resp,b,nil
}
func requestNoRedirect(method, target string, headers map[string]string, body []byte, connect, read time.Duration) (*http.Response, []byte, error) {
 req, err := http.NewRequest(method, target, bytes.NewReader(body))
 if err != nil { return nil, nil, err }
 for key, value := range headers { req.Header.Set(key, value) }
 client := *newHTTPClient(connect, read)
 client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
 resp, err := client.Do(req)
 if err != nil { return nil, nil, err }
 data, readErr := io.ReadAll(resp.Body)
 _ = resp.Body.Close()
 if readErr != nil { return resp, data, readErr }
 return resp, data, nil
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
 label = strings.TrimSpace(label)
 if label == "" { return "" }
 separators := regexp.MustCompile(`[\s,;|/]+`)
 tokens := separators.Split(label, -1)
 alpha3Pattern := regexp.MustCompile("^([A-Z]{3})([^A-Za-z]|$)")
 alpha2Pattern := regexp.MustCompile("^([A-Z]{2})([^A-Za-z]|$)")
 leadingNoise := regexp.MustCompile("^[\\d\\s\\-_.|#]+")
 chineseRuns := regexp.MustCompile("[\\x{4e00}-\\x{9fff}（）()]+")
 for _, token := range tokens {
  cleaned := leadingNoise.ReplaceAllString(strings.TrimSpace(token), "")
  if match := alpha3Pattern.FindStringSubmatch(cleaned); len(match) > 1 {
   if code := alpha3ToAlpha2[match[1]]; code != "" { return code }
  }
  if match := alpha2Pattern.FindStringSubmatch(cleaned); len(match) > 1 {
   if codeSet[match[1]] { return match[1] }
  }
 }
 for _, token := range tokens {
  cleaned := leadingNoise.ReplaceAllString(token, "")
  for _, candidate := range chineseRuns.FindAllString(cleaned, -1) {
   if code := cnToCode[candidate]; code != "" { return code }
  }
 }
 indicators := []rune{}
 for _, r := range label {
  if r >= 0x1F1E6 && r <= 0x1F1FF { indicators = append(indicators, r) }
 }
 if len(indicators) >= 2 && len(indicators)%2 == 0 {
  first := indicators[0] - 0x1F1E6
  second := indicators[1] - 0x1F1E6
  if first <= 25 && second <= 25 { return string([]rune{'A' + first, 'A' + second}) }
 }
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
   resp,b,e:=request("GET",s.URL,nil,nil,duration("FETCH_CONNECT_TIMEOUT",3),duration("FETCH_TIMEOUT",3))
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
func writeCache(path string, rows map[string]string) {
 keys := make([]string, 0, len(rows))
 for key := range rows { ip := strings.SplitN(key, ":", 2)[0]; if net.ParseIP(ip) != nil { keys = append(keys, key) } }
 sort.Slice(keys, func(i, j int) bool {
  ipI := strings.SplitN(keys[i], ":", 2)[0]
  ipJ := strings.SplitN(keys[j], ":", 2)[0]
  a, b := net.ParseIP(ipI), net.ParseIP(ipJ)
  if a != nil && b != nil { return bytes.Compare(a.To16(), b.To16()) < 0 }
  if a != nil { return true }
  if b != nil { return false }
  return keys[i] < keys[j]
 })
 var out strings.Builder
 for _, key := range keys { out.WriteString(key + "#" + rows[key] + "\n") }
 _ = os.WriteFile(path, []byte(out.String()), 0644)
}
type ipInfoToken struct {
 token string
 failures int
 cooldownUntil time.Time
 lastRequest time.Time
 rateMu sync.Mutex
}
type ipInfoTokenPool struct {
 mu sync.Mutex
 tokens []*ipInfoToken
 current int
 failureThreshold int
 minInterval time.Duration
}
func newIPInfoTokenPool(tokens []string) *ipInfoTokenPool {
 states := make([]*ipInfoToken, 0, len(tokens))
 for _, token := range tokens { states = append(states, &ipInfoToken{token: token}) }
 threshold := integer("TOKEN_FAILURE_THRESHOLD")
 if threshold < 2 { threshold = 2 }
 return &ipInfoTokenPool{tokens: states, failureThreshold: threshold, minInterval: time.Duration(number("IP_CALIBRATION_MIN_INTERVAL") * float64(time.Second))}
}
func (p *ipInfoTokenPool) acquire() (int, *ipInfoToken) {
 p.mu.Lock()
 defer p.mu.Unlock()
 now := time.Now()
 for i := 0; i < len(p.tokens); i++ {
  p.current = (p.current + 1) % len(p.tokens)
  state := p.tokens[p.current]
  if state.failures < p.failureThreshold && !state.cooldownUntil.After(now) { return p.current, state }
 }
 best := 0
 for i := 1; i < len(p.tokens); i++ { if p.tokens[i].failures < p.tokens[best].failures { best = i } }
 p.current = best
 return best, p.tokens[best]
}
func (p *ipInfoTokenPool) rateLimit(state *ipInfoToken) {
 state.rateMu.Lock()
 defer state.rateMu.Unlock()
 if !state.lastRequest.IsZero() {
  wait := state.lastRequest.Add(p.minInterval).Sub(time.Now())
  if wait > 0 { time.Sleep(wait) }
 }
 state.lastRequest = time.Now()
}
func (p *ipInfoTokenPool) cool(idx int, d time.Duration, failed bool) {
 p.mu.Lock()
 defer p.mu.Unlock()
 state := p.tokens[idx]
 state.cooldownUntil = time.Now().Add(d)
 if failed { state.failures++ }
}
func (p *ipInfoTokenPool) markSuccess(idx int) {
 p.mu.Lock()
 p.tokens[idx].failures = 0
 p.mu.Unlock()
}
func queryIPInfo(ip string, pool *ipInfoTokenPool) string {
 if pool == nil || len(pool.tokens) == 0 { return "" }
 attempts := max(3, min(len(pool.tokens), 10))
 for attempt := 0; attempt < attempts; attempt++ {
  idx, state := pool.acquire()
  pool.rateLimit(state)
  target := "https://ipinfo.io/" + url.PathEscape(ip) + "/json?token=" + url.QueryEscape(state.token)
  resp, body, err := request("GET", target, nil, nil, 5*time.Second, 5*time.Second)
  if err != nil { pool.cool(idx, 300*time.Millisecond, false); continue }
  if resp.StatusCode == http.StatusTooManyRequests { pool.cool(idx, time.Second, false); continue }
  if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden { pool.cool(idx, 0, true); continue }
  if resp.StatusCode < 200 || resp.StatusCode >= 300 { pool.cool(idx, 300*time.Millisecond, false); continue }
  var data map[string]any
  if json.Unmarshal(body, &data) != nil { pool.cool(idx, 300*time.Millisecond, false); continue }
  pool.markSuccess(idx)
  country, _ := data["country"].(string)
  if country == "" || country == "Unknown" { continue }
  parts := []string{country}
  for _, key := range []string{"region", "city"} {
   if value, ok := data[key].(string); ok && strings.TrimSpace(value) != "" && value != "Unknown" { parts = append(parts, strings.TrimSpace(value)) }
  }
  org, _ := data["org"].(string)
  asn, isp := "Unknown", "Unknown"
  if strings.HasPrefix(org, "AS") {
   fields := strings.Fields(org)
   if len(fields) > 0 { asn = fields[0] }
   if strings.Contains(org, " ") { isp = strings.TrimSpace(strings.SplitN(org, " ", 2)[1]) } else { isp = org }
  } else if org != "" && org != "Unknown" {
   isp = org
  }
  if asn != "Unknown" { parts = append(parts, asn) }
  if isp != "Unknown" { parts = append(parts, isp) }
  return strings.Join(parts, " ")
 }
 return ""
}

func calibrate(nodes []string) {
 if !boolean("IP_CALIBRATION_ENABLED"){logf("IP 地区校准已禁用，跳过。\n");return}
 tokenPath:=filepath.Join(root,str("IP_CALIBRATION_TOKEN_FILE"));cachePath:=filepath.Join(root,str("IP_CALIBRATION_CACHE_FILE"));cache:=readCache(cachePath)
 for i,n:=range nodes {ipp:=strings.SplitN(n,"#",2)[0];if tag:=cache[ipp];tag!=""{nodes[i]=ipp+"#"+strings.Fields(tag)[0]}}
 tokens:=loadLines(tokenPath);if len(tokens)==0{logf("valid_tokens.txt 为空，IP 地区校准跳过。\n");return}
 logf("正在进行 token 有效性校验...\n")
 workers := integer("IP_CALIBRATION_CONCURRENCY")
 if workers < 1 { workers = 1 }
 tokenOK := runPool(tokens, workers, func(t string) bool {
  u := "https://ipinfo.io/json?token=" + url.QueryEscape(t)
  resp, b, e := request("GET", u, nil, nil, 5*time.Second, 5*time.Second)
  if e != nil || resp.StatusCode != http.StatusOK { return false }
  var d map[string]any
  if json.Unmarshal(b, &d) != nil { return false }
  country, _ := d["country"].(string)
  return country != ""
 }, "Token 校验")
 valid := make([]string, 0, len(tokens))
 for i, ok := range tokenOK { if ok { valid = append(valid, tokens[i]) } }
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
  tokenPool := newIPInfoTokenPool(valid)
  for i:=0;i<workers;i++ {wg.Add(1);go func(){defer wg.Done();for ip:=range jobs {tag:=queryIPInfo(ip,tokenPool);results<-result{ip,tag}}}()}
  go func(){for _,ip:=range ips{jobs<-ip};close(jobs);wg.Wait();close(results)}()
  successCount := 0
  for r:=range results {atomic.AddInt64(&done,1);if r.tag!=""{successCount++;for _,ipp:=range ipports[r.ip]{cache[ipp]=r.tag}};logf("\r[%d/%d] 地区校准...",atomic.LoadInt64(&done),len(ips))}
  logf("\n")
  if len(ips) > 0 && successCount == 0 { logf("警告：所有 IP 地区校准失败，可能所有 token 均已失效。\n"); notify("IP地区校准：所有IP查询均失败，可能所有token均已失效。", "IP校准全失败") }
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
  if e==nil&&resp.StatusCode==200 {var d map[string]any;_ = json.Unmarshal(b,&d);if d["success"]==true {stack:=fmt.Sprint(d["inferred_stack"]);if stack=="<nil>"{stack="unknown"};var probe map[string]any;if p,ok:=d["probe_results"].(map[string]any);ok {probe,_=p["ipv6"].(map[string]any);if len(probe)==0{probe,_=p["ipv4"].(map[string]any)}};exit,_:=probe["exit"].(map[string]any);return AvailabilityResult{node,true,stack,exit}}}
  if i<maxAttempts-1{sleepSec("AVAILABILITY_INNER_RETRY_DELAY")}
 }
 return AvailabilityResult{Node:node,OK:false,Stack:"unknown"}
}
func httpTest(node string) HTTPResult {
 m:=ipPortPattern.FindStringSubmatch(node);if len(m)!=3{return HTTPResult{Node:node}}
 target:="http://"+net.JoinHostPort(m[1],m[2])+"/cdn-cgi/trace";method:=strings.ToUpper(str("HTTP_TEST_METHOD"));if method!="HEAD"{method="GET"}
 n:=max(3,integer("HTTP_JITTER_SAMPLES"));lats:=[]float64{}
 for i:=0;i<n;i++ {start:=time.Now();resp,_,e:=requestNoRedirect(method,target,map[string]string{"User-Agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"},nil,duration("HTTP_TEST_CONNECT_TIMEOUT",3),duration("HTTP_TEST_TIMEOUT",3))
  if e!=nil{return HTTPResult{Node:node,Server:"connection_error"}};lat:=float64(time.Since(start).Microseconds())/1000
  if resp.StatusCode!=400{return HTTPResult{Node:node,Server:fmt.Sprintf("status_%d",resp.StatusCode)}}
  server:=resp.Header.Get("Server");if !strings.HasPrefix(strings.ToLower(server),"cloudflare"){return HTTPResult{Node:node,Server:server}}
  lats=append(lats,lat)
 }
 var mean float64;for _,v:=range lats{mean+=v};mean/=float64(len(lats));var variance float64;for _,v:=range lats{variance+=(v-mean)*(v-mean)};variance/=float64(len(lats))
 maxlat:=0.0;for _,v:=range lats{if v>maxlat{maxlat=v}}
 return HTTPResult{Node:node,OK:true,Server:"cloudflare",Latency:maxlat,Jitter:math.Sqrt(variance)}
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
func syncSetting(script, key string) string {
 pattern := regexp.MustCompile("(?m)^\\s*(?:\\$)?" + regexp.QuoteMeta(key) + "\\s*=\\s*[\"']([^\"']*)[\"']")
 match := pattern.FindStringSubmatch(script)
 if len(match) < 2 { return "" }
 return strings.TrimSpace(match[1])
}

func runSyncCommand(ctx context.Context, args []string, token string) error {
 cmd := exec.CommandContext(ctx, args[0], args[1:]...)
 cmd.Dir = root
 output, err := cmd.CombinedOutput()
 safeOutput := strings.ReplaceAll(string(output), token, "***")
 if strings.TrimSpace(safeOutput) != "" { logf("%s\n", strings.TrimSpace(safeOutput)) }
 return err
}

func syncGitHub() {
 scriptName := "git_sync.sh"
 if runtime.GOOS == "windows" { scriptName = "git_sync.ps1" }
 scriptPath := filepath.Join(root, scriptName)
 data, err := os.ReadFile(scriptPath)
 if err != nil { logf("未找到 %s，跳过 GitHub 同步。\n", scriptName); return }
 source := string(data)
 token := syncSetting(source, "github_token")
 username := syncSetting(source, "github_username")
 repo := syncSetting(source, "repo_name")
 branch := syncSetting(source, "branch")
 if token == "" || username == "" || repo == "" || branch == "" {
  logf("%s 中 GitHub 同步配置不完整，跳过同步。\n", scriptName)
  return
 }
 pushURL := "https://" + token + "@github.com/" + username + "/" + repo + ".git"
 for attempt := 1; attempt <= integer("GITHUB_SYNC_MAX_RETRIES"); attempt++ {
  logf("\n正在同步到 GitHub (尝试 %d/%d)...\n", attempt, integer("GITHUB_SYNC_MAX_RETRIES"))
  ctx, cancel := context.WithTimeout(context.Background(), duration("GIT_SYNC_PROCESS_TIMEOUT", 180))
  _ = runSyncCommand(ctx, []string{"git", "pull", "origin", branch}, token)
  _ = runSyncCommand(ctx, []string{"git", "add", "ip.txt"}, token)
  commitMessage := "Update ip.txt on " + time.Now().Format("2006-01-02 15:04:05")
  _ = runSyncCommand(ctx, []string{"git", "commit", "-m", commitMessage}, token)
  pushErr := runSyncCommand(ctx, []string{"git", "push", pushURL, branch, "--force"}, token)
  cancel()
  if pushErr == nil {
   logf("已自动推送到 GitHub。\n")
   return
  }
  logf("GitHub 推送失败：%v\n", pushErr)
  if attempt < integer("GITHUB_SYNC_MAX_RETRIES") { sleepSec("GITHUB_SYNC_RETRY_DELAY") }
 }
 notify("GitHub 推送失败，请检查网络或仓库状态。", "GitHub 推送失败")
 logf("已尝试 %d 次推送，均失败，请检查网络或 GitHub 仓库状态。\n", integer("GITHUB_SYNC_MAX_RETRIES"))
}

func riskScore(v any) float64 {
 if v == nil { return 0 }
 s := strings.TrimSpace(fmt.Sprint(v))
 if m := regexp.MustCompile(`([0-9.]+)\s*\(([^)]+)\)`).FindStringSubmatch(s); len(m) > 1 {
  n, _ := strconv.ParseFloat(m[1], 64)
  return n
 }
 n, _ := strconv.ParseFloat(s, 64)
 return n
}

func getIPRiskLevel(ip string) string {
 target := "https://api.ipapi.is/?q=" + url.QueryEscape(ip)
 resp, body, err := request("GET", target, nil, nil, 5*time.Second, 10*time.Second)
 if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 { return "未知" }
 var data map[string]any
 if json.Unmarshal(body, &data) != nil { return "未知" }
 company, _ := data["company"].(map[string]any)
 asn, _ := data["asn"].(map[string]any)
 var companyScore, asnScore any
 if company != nil { companyScore = company["abuser_score"] }
 if asn != nil { asnScore = asn["abuser_score"] }
 score := ((riskScore(companyScore) + riskScore(asnScore)) / 2) * 5
 for _, key := range []string{"is_crawler", "is_proxy", "is_vpn", "is_tor", "is_abuser"} {
  if v, ok := data[key].(bool); ok && v { score += 0.15 }
 }
 if v, ok := data["is_bogon"].(bool); ok && v { score += 1 }
 percentage := score * 100
 switch {
 case percentage >= 100: return "极度危险"
 case percentage >= 20: return "高风险"
 case percentage >= 5: return "轻微风险"
 case percentage >= 0.25: return "纯净"
 default: return "极度纯净"
 }
}

func updateDNS(nodes []string, bw []BWResult, av map[string]AvailabilityResult, lat, httpLat, jitter, speed map[string]float64) {
 if !boolean("CF_ENABLED") { logf("Cloudflare DNS 批量更新未启用。\n"); return }
 recordType := strings.ToUpper(str("DNS_RECORD_TYPE"))
 if recordType != "A" && recordType != "TXT" { logf("不支持的 DNS_RECORD_TYPE: %s，已跳过 DNS 更新。\n", recordType); return }
 targetCount := integer("DNS_UPDATE_TARGET_COUNT")
 blockedSet := map[string]bool{}
 if boolean("FILTER_BLOCKED_COUNTRIES_ENABLED") {
  for _, country := range stringsVal("BLOCKED_COUNTRIES") { blockedSet[strings.ToUpper(country)] = true }
 }
 riskMap := map[string]string{}
 if boolean("DNS_IP_RISK_FILTER_ENABLED") && len(bw) > 0 {
  ipsSeen := map[string]bool{}
  ips := []string{}
  for _, item := range bw {
   m := ipPortPattern.FindStringSubmatch(item.Node)
   if len(m) == 3 && !ipsSeen[m[1]] { ipsSeen[m[1]] = true; ips = append(ips, m[1]) }
  }
  workers := min(max(1, integer("FALLBACK_WORKERS")), max(1, len(ips)))
  levels := runPool(ips, workers, func(ip string) string { return getIPRiskLevel(ip) }, "IP 风险查询")
  for i, ip := range ips { riskMap[ip] = levels[i] }
 }
 riskOrder := map[string]int{"极度纯净": 0, "纯净": 1, "轻微风险": 2, "高风险": 3, "极度危险": 4}
 maxRisk := integer("DNS_IP_RISK_MAX_LEVEL")
 maxRiskName := str("DNS_IP_RISK_MAX_LEVEL")
 if v, ok := riskOrder[maxRiskName]; ok { maxRisk = v } else { maxRisk = 2 }
 content := []string{}
 seen := map[string]bool{}
 riskFallback := []string{}
 for _, item := range bw {
  m := ipPortPattern.FindStringSubmatch(item.Node)
  if len(m) != 3 { continue }
  ip, port := m[1], m[2]
  if recordType == "A" && port != "443" { continue }
  if boolean("FILTER_IPV6_AVAILABILITY") {
   if a, ok := av[item.Node]; ok && a.Stack == "ipv6_only" { continue }
  }
  country := ""
  if p := strings.SplitN(item.Node, "#", 2); len(p) == 2 {
   fields := strings.Fields(p[1]); if len(fields) > 0 { country = strings.ToUpper(fields[0]) }
  }
  if blockedSet[country] { continue }
  if boolean("DNS_IP_RISK_FILTER_ENABLED") {
   value := ip
   if recordType == "TXT" { value = ip + ":" + port }
   if !seen["riskfallback:"+value] { riskFallback = append(riskFallback, value); seen["riskfallback:"+value] = true }
   level := riskMap[ip]
   rank, known := riskOrder[level]
   if !known || rank > maxRisk { continue }
  }
  value := ip
  if recordType == "TXT" { value = ip + ":" + port }
  if !seen[value] { content = append(content, value); seen[value] = true }
  if len(content) >= targetCount { break }
 }
 if len(content) == 0 && boolean("DNS_IP_RISK_FILTER_ENABLED") && len(riskFallback) > 0 {
  notify("风险等级检测全部失败或候选节点均超出允许风险等级，已回退到无风险等级过滤的候选列表。", "风险等级检测全部失败")
  for _, value := range riskFallback {
   if !seen["fallback:"+value] { content = append(content, value); seen["fallback:"+value] = true }
   if len(content) >= targetCount { break }
  }
 }
 if len(content) == 0 && recordType == "A" {
  for _, node := range nodes {
   m := ipPortPattern.FindStringSubmatch(node)
   if len(m) == 3 && !seen[m[1]] { content = append(content, m[1]); seen[m[1]] = true; if len(content) >= targetCount { break } }
  }
 }
 if len(content) == 0 {
  logf("没有可用的 IP 用于 DNS 更新，跳过。\n")
  notify("没有可用的 IP 用于 DNS 更新，跳过。", "DNS 更新跳过")
  return
 }
 logf("准备更新 Cloudflare DNS：%d 条记录，类型 %s\n", len(content), recordType)
 headers := map[string]string{"Authorization": "Bearer " + str("CF_API_TOKEN"), "Content-Type": "application/json"}
 for attempt := 1; attempt <= integer("DNS_UPDATE_MAX_RETRIES"); attempt++ {
  base := "https://api.cloudflare.com/client/v4/zones/" + url.PathEscape(str("CF_ZONE_ID")) + "/dns_records"
  target := base + "?type=" + recordType + "&name=" + url.QueryEscape(str("CF_DNS_RECORD_NAME"))
  resp, body, err := request("GET", target, headers, nil, duration("CF_DNS_CONNECT_TIMEOUT", 3), duration("CF_DNS_READ_TIMEOUT", 3))
  if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
   var data map[string]any
   _ = json.Unmarshal(body, &data)
   if data["success"] == true {
    records, _ := data["result"].([]any)
    deletes := []any{}
    for _, item := range records { if record, ok := item.(map[string]any); ok { if id, exists := record["id"]; exists { deletes = append(deletes, map[string]any{"id": id}) } } }
    posts := []any{}
    for _, value := range content {
     record := map[string]any{"name": str("CF_DNS_RECORD_NAME"), "type": recordType, "content": value, "ttl": integer("CF_TTL")}
     if recordType == "A" { record["proxied"] = boolean("CF_PROXIED") }
     posts = append(posts, record)
    }
    payload, _ := json.Marshal(map[string]any{"deletes": deletes, "posts": posts})
    batchResp, batchBody, batchErr := request("POST", base+"/batch", headers, payload, duration("CF_DNS_CONNECT_TIMEOUT", 3), duration("CF_DNS_READ_TIMEOUT", 3))
    var batchResult map[string]any
    if batchErr == nil { _ = json.Unmarshal(batchBody, &batchResult) }
    if batchErr == nil && batchResp.StatusCode >= 200 && batchResp.StatusCode < 300 && batchResult["success"] == true {
     logf("Cloudflare DNS 批量更新成功！共 %d 条记录。\n", len(content))
     return
    }
    if batchErr != nil { err = batchErr } else { err = fmt.Errorf("批量更新失败: HTTP %d", batchResp.StatusCode) }
   } else { err = errors.New("查询 DNS 记录失败") }
  } else if err == nil { err = fmt.Errorf("HTTP %d", resp.StatusCode) }
  logf("[尝试 %d/%d] DNS 更新出错: %v\n", attempt, integer("DNS_UPDATE_MAX_RETRIES"), err)
  if attempt < integer("DNS_UPDATE_MAX_RETRIES") { sleepSec("DNS_UPDATE_RETRY_DELAY") }
 }
 notify("Cloudflare DNS 更新失败，请检查配置和网络。", "DNS 更新失败")
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
 bwBySpeed:=append([]BWResult(nil),bw...);speed:=map[string]float64{};selected:=[]string{}
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
 updateDNS(selected,bwBySpeed,avMap,latency,httpLat,jitter,speed)
 syncGitHub()
}
