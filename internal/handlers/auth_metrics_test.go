package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"auth-service/internal/metrics"
)

func setupAuthMetricsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	registry := prometheus.NewRegistry()
	metricsCollector := metrics.NewWithRegistry("auth-service", registry, registry)
	handler := NewAuthHandler(nil, "test-secret", nil, metricsCollector)

	router := gin.New()
	router.Use(metricsCollector.Middleware())
	router.GET("/metrics", gin.WrapH(metricsCollector.Handler()))
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/register", handler.Register)

	return router
}

func TestAuthLoginFailedMetrics(t *testing.T) {
	router := setupAuthMetricsRouter()

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not-json"))
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", loginResp.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, metricsReq)
	body := metricsResp.Body.String()
	if !strings.Contains(body, `auth_logins_total{status="failed"}`) {
		t.Fatalf("expected auth_logins_total failed metric in /metrics output")
	}
}

func TestAuthRegisterFailedMetrics(t *testing.T) {
	router := setupAuthMetricsRouter()

	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`{}`))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp := httptest.NewRecorder()
	router.ServeHTTP(registerResp, registerReq)
	if registerResp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", registerResp.Code)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, metricsReq)
	body := metricsResp.Body.String()
	if !strings.Contains(body, `auth_registers_total{status="failed"}`) {
		t.Fatalf("expected auth_registers_total failed metric in /metrics output")
	}
}
