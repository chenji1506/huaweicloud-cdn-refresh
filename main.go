package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	username    = flag.String("username", "refresh_cdn", "华为云 IAM 用户名 (Huawei cloud user name)")
	password    = flag.String("password", "G1pxvRkGt", "华为云 IAM 密码 (Huawei cloud user password)")
	refreshType = flag.String("refreshtype", "directory", "刷新类型, 可选值: file(文件刷新) / directory(目录刷新)")
	urls        = flag.String("refreurls", "", "要刷新的 URL, 多个用逗号分隔, 例: http://a.com/1.png,http://a.com/2.png")
	domain      = flag.String("domain", "default", "华为云 IAM 认证域名称 (auth domain name)")
	ak          = flag.String("ak", "", "访问密钥 AK/SK 中的 Access Key (与 -sk 同时指定后启用 AK/SK 签名认证, 免用户名密码)")
	sk          = flag.String("sk", "", "访问密钥 AK/SK 中的 Secret Key (与 -ak 同时指定后启用 AK/SK 签名认证)")
)

var token string

func newClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

func getToken() {
	body := `{"auth": {` +
		`     "identity": {` +
		`       "methods": ["password"],` +
		`       "password": {` +
		`         "user": {` +
		`           "name": "` + *username + `",` +
		`           "password": "` + *password + `",` +
		`           "domain": {` +
		`             "name": "` + *domain + `"` +
		`           }` +
		`         }` +
		`       }` +
		`     },` +
		`     "scope": {` +
		`       "domain": {` +
		`         "name": "` + *domain + `"` +
		`       }` +
		`     }` +
		`   } }`

	req, err := http.NewRequest("POST", "https://cdn.myhwclouds.com/v3/auth/tokens",
		bytes.NewBuffer([]byte(body)))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Content-Type", "application/json;charset=utf8")

	resp, err := newClient().Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	token = resp.Header.Get("X-Subject-Token")
	if token == "" {
		fmt.Println("token is null, auth response:", string(data))
	}
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// SDK-HMAC-SHA256 签名 (华为云 AK/SK 认证)
func signRequest(req *http.Request, body []byte) {
	sdkDate := time.Now().UTC().Format("20060102T150405Z")
	host := req.URL.Host
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	} else if !strings.HasSuffix(canonicalURI, "/") {
		// 华为云网关签名规范要求 CanonicalURI 以 "/" 结尾
		canonicalURI += "/"
	}
	canonicalQuery := req.URL.RawQuery // 本工具无 query, 保留通用性

	signedHeaders := "host;x-sdk-date"
	canonicalHeaders := "host:" + host + "\n" + "x-sdk-date:" + sdkDate + "\n"

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		sha256Hex(body),
	}, "\n")

	stringToSign := "SDK-HMAC-SHA256\n" + sdkDate + "\n" + sha256Hex([]byte(canonicalRequest))

	mac := hmac.New(sha256.New, []byte(*sk))
	mac.Write([]byte(stringToSign))
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Sdk-Date", sdkDate)
	req.Header.Set("Authorization",
		"SDK-HMAC-SHA256 Access="+*ak+", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func useAKSK() bool {
	return *ak != "" && *sk != ""
}

func buildURLArray(raw string) string {
	parts := strings.Split(raw, `","`)
	if len(parts) == 1 {
		parts = strings.Split(raw, ",")
	}
	var items []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.Trim(p, `"`))
		if p == "" {
			continue
		}
		b, _ := json.Marshal(p)
		items = append(items, string(b))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func refreshtasks() {
	urlArray := buildURLArray(*urls)
	if urlArray == "[]" {
		fmt.Println("empty url")
		return
	}
	body := []byte(`{"refreshTask":{` +
		`         "type":"` + *refreshType + `",` +
		`         "urls":` + urlArray +
		`     } }`)

	req, err := http.NewRequest("POST", "https://cdn.myhwclouds.com/v1.0/cdn/refreshtasks",
		bytes.NewBuffer(body))
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Content-Type", "application/json;charset=utf8")
	if useAKSK() {
		signRequest(req, body)
	} else {
		req.Header.Set("X-Auth-Token", token)
	}

	resp, err := newClient().Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(data))
}

func main() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "华为云 CDN 缓存刷新工具 (cache refresh via cdn.myhwclouds.com)\n\n")
		fmt.Fprintf(out, "用法: %s [参数]\n\n参数:\n", os.Args[0])
		printFlag := func(name string) {
			f := flag.Lookup(name)
			if f.DefValue != "" {
				fmt.Fprintf(out, "  -%s\n    \t%s (默认 %q)\n", f.Name, f.Usage, f.DefValue)
			} else {
				fmt.Fprintf(out, "  -%s\n    \t%s\n", f.Name, f.Usage)
			}
		}
		for _, name := range []string{"ak", "sk", "username", "password", "domain", "refreshtype", "refreurls"} {
			printFlag(name)
		}
		fmt.Fprintf(out, "\n示例:\n")
		fmt.Fprintf(out, "  # 刷新单个文件\n  %s -refreshtype file -refreurls \"http://www.example.com/a.png\"\n", os.Args[0])
		fmt.Fprintf(out, "  # 刷新多个文件\n  %s -refreshtype file -refreurls \"http://www.example.com/a.png,http://www.example.com/b.png\"\n", os.Args[0])
		fmt.Fprintf(out, "  # 刷新目录\n  %s -refreshtype directory -refreurls \"http://www.example.com/dir/\"\n", os.Args[0])
		fmt.Fprintf(out, "  # 指定账号和认证域 (Token 认证)\n  %s -username user1 -password pass1 -domain mydomain -refreshtype file -refreurls \"http://www.example.com/a.png\"\n", os.Args[0])
		fmt.Fprintf(out, "  # 使用 AK/SK 签名认证 (无需用户名密码/Token)\n  %s -ak YOUR_AK -sk YOUR_SK -refreshtype file -refreurls \"http://www.example.com/a.png\"\n", os.Args[0])
	}
	flag.Parse()
	if *urls == "" {
		fmt.Println("urls is null")
		return
	}
	if *refreshType != "file" && *refreshType != "directory" {
		fmt.Println("refre type value  can be file or directory")
		return
	}
	if useAKSK() {
		refreshtasks()
		return
	}
	getToken()
	if token == "" {
		return
	}
	refreshtasks()
}
