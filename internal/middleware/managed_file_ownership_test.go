package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func runManagedResourceGuard(t *testing.T, activeTenant uint64, role types.TenantRole, userID string, resource OwnershipResource, policy ManagedResourcePolicy, apiKey bool) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Tenant: &config.TenantConfig{FileOwnershipTenantID: 42}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, activeTenant)
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
		if apiKey {
			ctx = types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{KeyID: 1})
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	guard := RequireManagedTenantResource(func(*gin.Context) (OwnershipResource, error) {
		return resource, nil
	}, policy, cfg)
	r.PUT("/resource", guard, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/resource", nil))
	return rec.Code
}

func TestManagedDocumentMutationPolicyHTTP(t *testing.T) {
	tests := []struct {
		name         string
		activeTenant uint64
		role         types.TenantRole
		userID       string
		creatorID    string
		want         int
	}{
		{name: "contributor owns file", activeTenant: 42, role: types.TenantRoleContributor, userID: "alice", creatorID: "alice", want: http.StatusNoContent},
		{name: "foreign owner", activeTenant: 42, role: types.TenantRoleContributor, userID: "alice", creatorID: "bob", want: http.StatusForbidden},
		{name: "null owner", activeTenant: 42, role: types.TenantRoleContributor, userID: "alice", want: http.StatusForbidden},
		{name: "viewer creator cannot bypass", activeTenant: 42, role: types.TenantRoleViewer, userID: "alice", creatorID: "alice", want: http.StatusForbidden},
		{name: "admin manages foreign file", activeTenant: 42, role: types.TenantRoleAdmin, userID: "admin", creatorID: "bob", want: http.StatusNoContent},
		{name: "forged active tenant cannot use role", activeTenant: 7, role: types.TenantRoleAdmin, userID: "admin", creatorID: "admin", want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := runManagedResourceGuard(t, test.activeTenant, test.role, test.userID, OwnershipResource{TenantID: 42, CreatorID: test.creatorID}, ManagedResourceDocumentOwner, false)
			if got != test.want {
				t.Fatalf("status = %d, want %d", got, test.want)
			}
		})
	}
}

func TestManagedKBAndUploadPoliciesHTTP(t *testing.T) {
	if got := runManagedResourceGuard(t, 42, types.TenantRoleContributor, "alice", OwnershipResource{TenantID: 42}, ManagedResourceContributorCreate, false); got != http.StatusNoContent {
		t.Fatalf("document upload status = %d", got)
	}
	if got := runManagedResourceGuard(t, 42, types.TenantRoleContributor, "alice", OwnershipResource{TenantID: 42, ManagedAdminOnly: true}, ManagedResourceContributorCreate, false); got != http.StatusForbidden {
		t.Fatalf("FAQ/wiki upload status = %d, want 403", got)
	}
	if got := runManagedResourceGuard(t, 42, types.TenantRoleContributor, "alice", OwnershipResource{TenantID: 42, CreatorID: "alice"}, ManagedResourceAdmin, false); got != http.StatusForbidden {
		t.Fatalf("managed KB creator lifecycle status = %d, want 403", got)
	}
}

func TestManagedPolicyLeavesLegacyAndAPIKeyAuthorityIntact(t *testing.T) {
	if got := runManagedResourceGuard(t, 7, types.TenantRoleContributor, "alice", OwnershipResource{TenantID: 7, CreatorID: "alice"}, ManagedResourceDocumentOwner, false); got != http.StatusNoContent {
		t.Fatalf("legacy owner status = %d", got)
	}
	if got := runManagedResourceGuard(t, 42, types.TenantRoleViewer, "system-42", OwnershipResource{TenantID: 42, CreatorID: "bob"}, ManagedResourceDocumentOwner, true); got != http.StatusNoContent {
		t.Fatalf("API key status = %d; route capability and KB scope remain authoritative", got)
	}
}
