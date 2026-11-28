// ArkPlayerPortal：ArkServerCommander 的玩家站。只读取 Commander 写好的快照，提供网页和只读接口
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TBro1998/ArkPlayerPortal/internal/auth"
	"github.com/TBro1998/ArkPlayerPortal/internal/server"
	"github.com/TBro1998/ArkPlayerPortal/internal/snapshot"
	"github.com/TBro1998/ArkPlayerPortal/web"
)

// version 构建时用 -ldflags "-X main.version=..." 注入
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(healthcheck())
		case "version":
			fmt.Println(version)
			return
		default:
			fmt.Fprintf(os.Stderr, "未知的子命令 %q（可用：healthcheck、version）\n", os.Args[1])
			os.Exit(2)
		}
	}
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func run() error {
	dataDir := env("PORTAL_DATA_DIR", "/data")
	listen := env("PORTAL_LISTEN", ":8080")
	trusted, err := server.ParseTrustedProxies(os.Getenv("PORTAL_TRUSTED_PROXIES"))
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := snapshot.NewStore(dataDir)
	go store.Watch(ctx, time.Second)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	login := &server.Login{
		Identity:  auth.NewIdentity(dataDir),
		Players:   auth.NewPlayers(dataDir),
		Arkfgf:    &auth.ArkfgfKeys{URL: os.Getenv("PORTAL_ARKFGF_KEYS_URL"), HTTP: httpClient},
		Steam:     &auth.Steam{HTTP: httpClient},
		PublicURL: os.Getenv("PORTAL_PUBLIC_URL"),
	}
	srv := &http.Server{
		Addr:              listen,
		Handler:           server.New(store, web.FS(), trusted).WithLogin(login).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second, // SSE 连接自己去掉写超时
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	errc := make(chan error, 1)
	go func() {
		log.Printf("ArkPlayerPortal %s 监听 %s，快照目录 %s", version, listen, dataDir)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// SSE 是长连接，等不完就直接关
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// healthcheck 镜像里没有 shell 和 wget，Docker 的 HEALTHCHECK 用这个子命令请求自己的 /healthz
func healthcheck() int {
	_, port, err := net.SplitHostPort(env("PORTAL_LISTEN", ":8080"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, resp.Status)
		return 1
	}
	return 0
}
