FROM node:20.20.0 AS node
WORKDIR /app/frontend
COPY frontend /app/frontend
RUN npm install --registry=https://registry.npmmirror.com && npm run build:pro&&rm -rf dist/log

FROM golang:1.25-alpine AS gobuilder
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 \
    GOPATH=/root/gopath \
    GOPROXY=https://goproxy.cn,direct \
    GO111MODULE='on' \
    GIT_TERMINAL_PROMPT=1
WORKDIR /app

# 安装必要的依赖
RUN apk add --no-cache make git

# 复制 go.mod 和 go.sum
COPY . /app
COPY --from=node /app/frontend/dist/assets /app/web/dist/assets
COPY --from=node /app/frontend/dist/index.html /app/web/dist/index.html

# 构建应用
RUN wget https://github.com/upx/upx/releases/download/v5.1.1/upx-5.1.1-amd64_linux.tar.xz -O /tmp/upx-5.1.1-amd64_linux.tar.xz && \
    tar -xf /tmp/upx-5.1.1-amd64_linux.tar.xz -C /tmp && \
    cp /tmp/upx-5.1.1-amd64_linux/upx /usr/local/bin/upx && \
    go mod download && make build

# 生产镜像
FROM busybox


WORKDIR /app

# 从构建镜像复制二进制文件
COPY --from=gobuilder /app/bin/podcast .

ENTRYPOINT ["/app/podcast"]
# 启动应用
CMD [ "-f", "config/config.yaml"]
