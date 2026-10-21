package validator_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ngo/precedent_mcp/internal/env"
)

func ginContext(t *testing.T, body string, headers map[string]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, string(env.RouteMCP), strings.NewReader(body))
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	return c
}
