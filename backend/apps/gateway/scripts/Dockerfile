FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# 复制整个 backend workspace（含 go.work、apps/*、pkg/*），供 go.work 解析共享模块
COPY . .

# 同步 workspace 并下载依赖
RUN go work sync && go mod download

# 构建 gateway 聚合应用（入口 ./apps/gateway/cmd），单进程挂载 auth/platformadmin/tenantadmin
RUN go build -ldflags="-s -w" -o /app/gateway ./apps/gateway/cmd

FROM alpine:3.22

# 从构建阶段复制二进制
COPY --from=builder /app/gateway /app/gateway

# 从构建阶段复制运行配置与 OIDC 签名密钥
COPY --from=builder /build/apps/gateway/config/config.yaml       /app/config.yaml
COPY --from=builder /build/apps/gateway/config/oidc-dev-key.pem  /app/config/oidc-dev-key.pem

WORKDIR /app

EXPOSE 8100

ENTRYPOINT ["/app/gateway"]
