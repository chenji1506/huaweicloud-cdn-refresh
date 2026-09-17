# huaweicloud-cdn-refresh

华为云 CDN 缓存刷新命令行工具，通过 `cdn.myhwclouds.com` 接口提交 URL/目录刷新任务。

支持两种认证方式：

1. **Token 认证**：用户名 + 密码 + 认证域，先获取 IAM Token 再提交刷新任务
2. **AK/SK 签名认证**（SDK-HMAC-SHA256）：直接对请求签名，无需获取 Token

## 编译

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o huaweicloud-cdn-refresh .

# Windows
go build -o huaweicloud-cdn-refresh.exe .
```

## 参数

| 参数 | 说明 |
|---|---|
| `-ak` | Access Key（与 `-sk` 同时指定后启用 AK/SK 签名认证） |
| `-sk` | Secret Key |
| `-username` | 华为云 IAM 用户名（Token 认证） |
| `-password` | 华为云 IAM 密码（Token 认证） |
| `-domain` | 华为云 IAM 认证域名称（Token 认证） |
| `-refreshtype` | 刷新类型：`file`（文件）/ `directory`（目录），默认 `directory` |
| `-refreurls` | 要刷新的 URL，多个用逗号分隔 |

各参数默认值可通过 `-h` 查看。

## 用法示例

```bash
# AK/SK 签名认证（推荐）
./huaweicloud-cdn-refresh -ak YOUR_AK -sk YOUR_SK \
    -refreshtype file \
    -refreurls "http://www.example.com/a.png,http://www.example.com/b.png"

# 刷新目录
./huaweicloud-cdn-refresh -ak YOUR_AK -sk YOUR_SK \
    -refreshtype directory \
    -refreurls "http://www.example.com/dir/"

# Token 认证（用户名/密码/认证域）
./huaweicloud-cdn-refresh -username user1 -password pass1 -domain mydomain \
    -refreshtype file \
    -refreurls "http://www.example.com/a.png"
```

成功时输出刷新任务响应 JSON（含 `refreshTaskID`），失败时输出错误信息。

## 说明

- AK/SK 模式下 `-username` / `-password` / `-domain` 参数无效
- 请求超时 60 秒
- AK/SK 签名遵循华为云网关规范（CanonicalURI 以 `/` 结尾）
