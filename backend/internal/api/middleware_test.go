package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"t11-unstoppables/internal/auth"
	"t11-unstoppables/internal/models"

	"github.com/gin-gonic/gin"
)

type middlewareUsers struct{}

func (middlewareUsers) CreateAccount(context.Context, string, string, string, string, string, string) (models.User, error) {
	return models.User{ID: "123e4567-e89b-12d3-a456-426614174000", Role: "rider"}, nil
}

func (middlewareUsers) CreateInvitation(context.Context, string, string, string, time.Time) error {
	return nil
}

func (middlewareUsers) UserByEmail(context.Context, string) (models.User, error) {
	return models.User{}, nil
}

func TestRequireAuthAndRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, err := auth.NewService(middlewareUsers{}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.RegisterRider(context.Background(), "Rider", "rider@example.com", "", "password-at-least-12")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/admin", RequireAuth(service), RequireRole("admin"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/account", RequireAuth(service), func(c *gin.Context) { c.String(http.StatusOK, c.GetString("userID")) })

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/account", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}
	account := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/account", nil)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(account, request)
	if account.Code != http.StatusOK || account.Body.String() != session.User.ID {
		t.Fatalf("expected authenticated account, got %d %q", account.Code, account.Body.String())
	}
	forbidden := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	router.ServeHTTP(forbidden, request)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", forbidden.Code)
	}
}