# 前端构建
FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 后端构建，网页打包进程序
FROM golang:1.27-alpine AS build
ARG GOPROXY
ARG VERSION=dev
WORKDIR /src
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/portal .

# 运行：只有一个静态程序，没有 shell，以非 root 用户运行
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/portal /portal
ENV PORTAL_DATA_DIR=/data PORTAL_LISTEN=:8080
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/portal", "healthcheck"]
ENTRYPOINT ["/portal"]
