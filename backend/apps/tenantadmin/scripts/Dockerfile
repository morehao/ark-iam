FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# 复制整个 backend workspace（含 go.work、apps/*、pkg/*），供 go.work 解析共享模块
COPY . .

# 同步 workspace 并下载依赖
RUN go work sync && go mod download

# 构建 tenantadmin 应用（入口 ./apps/tenantadmin/cmd）
RUN go build -ldflags="-s -w" -o /app/tenantadmin ./apps/tenantadmin/cmd

FROM alpine:3.22

# 从构建阶段复制二进制
COPY --from=builder /app/tenantadmin /app/tenantadmin

# 从构建阶段复制运行配置与 OIDC 签名密钥
COPY --from=builder /build/apps/tenantadmin/config/config.yaml       /app/config.yaml
COPY --from=builder /build/apps/tenantadmin/config/oidc-dev-key.pem  /app/config/oidc-dev-key.pem

WORKDIR /app

EXPOSE 8083

ENTRYPOINT ["/app/tenantadmin"]
