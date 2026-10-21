package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mail-api/internal/handler/email"
	"mail-api/internal/mailer"
	"mail-api/internal/mailqueue"
	"mail-api/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":9998"), "监听地址，例如 :9998")
	templatePath := flag.String("template", envOr("EMAIL_TEMPLATE", "email-template.html"), "HTML 邮件模板路径")
	configPath := flag.String("config", envOr("MAIL_CONFIG", "config.yaml"), "邮件服务配置文件")
	flag.Parse()

	sender, err := newTemplateSender(*templatePath, *configPath)
	if err != nil {
		slog.Error("mailer setup failed", "err", err)
		os.Exit(1)
	}

	jobs := mailqueue.New()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		jobs.Run(ctx, sender.Send)
	}()

	gin.SetMode(envOr("GIN_MODE", gin.ReleaseMode))

	engine := gin.New()
	engine.Use(gin.Recovery())
	router.Setup(engine, email.NewHandler(jobs).Post)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "template", *templatePath, "provider", sender.provider, "smtp", sender.host)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	serverErr := false
	select {
	case err := <-errCh:
		slog.Error("server failed", "err", err)
		serverErr = true
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	jobs.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "err", err)
		serverErr = true
	}
	stop()
	select {
	case <-workerDone:
	case <-time.After(8 * time.Second):
		slog.Error("mail queue stop timed out", "pending", jobs.Len())
	}
	if serverErr {
		os.Exit(1)
	}
}

type templateSender struct {
	mailer   *mailer.Mailer
	path     string
	provider string
	host     string
}

func (s templateSender) Send(ctx context.Context, to string) error {
	return mailer.SendTemplateFile(ctx, s.mailer, s.path, to)
}

func newTemplateSender(templatePath, configPath string) (templateSender, error) {
	cfg, err := mailer.LoadConfig(configPath)
	if err != nil {
		return templateSender{}, err
	}
	client, err := mailer.New(cfg)
	if err != nil {
		return templateSender{}, err
	}
	if _, err := os.Stat(templatePath); err != nil {
		return templateSender{}, fmt.Errorf("%w: %s", mailer.ErrTemplate, err.Error())
	}
	return templateSender{
		mailer:   client,
		path:     templatePath,
		provider: cfg.Provider,
		host:     cfg.Host,
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
