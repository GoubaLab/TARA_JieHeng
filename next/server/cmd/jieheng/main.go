package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"jieheng.local/tara/internal/assurance"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func locateRoot(explicit string) (string, error) {
	if explicit != "" {
		root, e := filepath.Abs(explicit)
		if e != nil {
			return "", e
		}
		if _, e = os.Stat(filepath.Join(root, "next", "data", "regulations", "manifest.json")); e != nil {
			return "", fmt.Errorf("--root 必须指向项目根目录: %w", e)
		}
		return root, nil
	}
	candidates := []string{}
	if p, e := os.Getwd(); e == nil {
		candidates = append(candidates, p)
	}
	if p, e := os.Executable(); e == nil {
		candidates = append(candidates, filepath.Dir(p))
	}
	for _, p := range candidates {
		for {
			if _, e := os.Stat(filepath.Join(p, "next", "data", "regulations", "manifest.json")); e == nil {
				return p, nil
			}
			parent := filepath.Dir(p)
			if parent == p {
				break
			}
			p = parent
		}
	}
	return "", fmt.Errorf("找不到项目目录，请使用 --root 指定")
}
func main() {
	if e := run(); e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
func run() error {
	port := flag.Int("port", 8790, "本机监听端口")
	database := flag.String("database", "", "SQLite 数据库路径")
	rootFlag := flag.String("root", "", "项目根目录")
	enterprise := flag.Bool("enterprise", false, "启用企业账号、项目权限与 HTTPS 代理入口")
	publicOrigin := flag.String("public-origin", "", "企业 HTTPS 入口，例如 https://tara.example.internal")
	bootstrapAdmin := flag.String("bootstrap-admin", "", "首次创建企业管理员后退出")
	passwordFile := flag.String("bootstrap-password-file", "", "仅当前用户可读的初始口令文件")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		return fmt.Errorf("端口必须为 1–65535")
	}
	root, e := locateRoot(*rootFlag)
	if e != nil {
		return e
	}
	dbPath := *database
	if dbPath == "" {
		dbPath = filepath.Join(root, "next", "var", "assurance.sqlite3")
	}
	repo, e := assurance.NewRepository(dbPath)
	if e != nil {
		return e
	}
	defer repo.Close()
	if *bootstrapAdmin != "" {
		if *passwordFile == "" {
			return fmt.Errorf("请提供 --bootstrap-password-file")
		}
		info, e := os.Lstat(*passwordFile)
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return fmt.Errorf("初始口令文件须为普通文件且权限为 0600")
		}
		b, e := os.ReadFile(*passwordFile)
		if e != nil {
			return e
		}
		if e = repo.BootstrapAdmin(context.Background(), *bootstrapAdmin, strings.TrimRight(string(b), "\r\n")); e != nil {
			return e
		}
		fmt.Println("企业管理员已初始化；请妥善移除初始口令文件，再启动企业模式。")
		return nil
	}
	catalog, e := assurance.NewCatalog("", filepath.Join(root, "next", "data", "regulations"))
	if e != nil {
		return e
	}
	logger := log.New(os.Stderr, "jieheng ", log.LstdFlags|log.LUTC)
	address := fmt.Sprintf("127.0.0.1:%d", *port)
	listener, e := net.Listen("tcp", address)
	if e != nil {
		return e
	}
	defer listener.Close()
	var handler http.Handler
	if *enterprise {
		handler, e = assurance.NewEnterpriseHTTPHandler(repo, catalog, root, logger, *publicOrigin)
		if e != nil {
			return e
		}
	} else {
		if *publicOrigin != "" {
			return fmt.Errorf("--public-origin 仅用于企业模式")
		}
		handler = assurance.NewHTTPHandler(repo, catalog, root, logger)
	}
	defer handler.(*assurance.HTTPServer).Close()
	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", *port), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 180 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	fmt.Printf("界衡工作台 http://%s（Go）\n", server.Addr)
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopped)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case e = <-done:
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	case <-stopped:
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if e = server.Shutdown(ctx); e != nil {
			_ = server.Close()
			return e
		}
		e = <-done
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	}
}
