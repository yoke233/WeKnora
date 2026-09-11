package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func managedBatchContext(t *testing.T, role types.TenantRole, userID string) (*KnowledgeHandler, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(42))
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	c.Request = httptest.NewRequest("POST", "/knowledge/batch-delete", nil).WithContext(ctx)
	c.Set(types.TenantIDContextKey.String(), uint64(42))
	return &KnowledgeHandler{cfg: &config.Config{Tenant: &config.TenantConfig{FileOwnershipTenantID: 42}}}, c
}

func TestRequireManagedKnowledgeMutationsRejectsMixedBatch(t *testing.T) {
	h, c := managedBatchContext(t, types.TenantRoleContributor, "alice")
	items := []*types.Knowledge{
		{ID: "own", TenantID: 42, CreatorID: new("alice")},
		{ID: "foreign", TenantID: 42, CreatorID: new("bob")},
	}
	if err := h.requireManagedKnowledgeMutations(c, items); err == nil {
		t.Fatal("mixed ownership batch must be rejected before the caller performs side effects")
	}
}

func TestRequireManagedKnowledgeMutationsRoleAndNullEdges(t *testing.T) {
	tests := []struct {
		name    string
		role    types.TenantRole
		creator *string
		wantErr bool
	}{
		{name: "owner contributor", role: types.TenantRoleContributor, creator: new("alice")},
		{name: "viewer owner", role: types.TenantRoleViewer, creator: new("alice"), wantErr: true},
		{name: "null owner", role: types.TenantRoleContributor, wantErr: true},
		{name: "admin null owner", role: types.TenantRoleAdmin},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h, c := managedBatchContext(t, test.role, "alice")
			err := h.requireManagedKnowledgeMutations(c, []*types.Knowledge{{TenantID: 42, CreatorID: test.creator}})
			if (err != nil) != test.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestRequireManagedKnowledgeMutationsLeavesPersonalTenantUnchanged(t *testing.T) {
	h, c := managedBatchContext(t, types.TenantRoleViewer, "alice")
	if err := h.requireManagedKnowledgeMutations(c, []*types.Knowledge{{TenantID: 7}}); err != nil {
		t.Fatalf("personal tenant should retain legacy authorization: %v", err)
	}
}

func TestRequireManagedKBContentAccessKeepsPublicKBsAdminOnly(t *testing.T) {
	h, contributor := managedBatchContext(t, types.TenantRoleContributor, "alice")
	faq := &types.KnowledgeBase{TenantID: 42, Type: types.KnowledgeBaseTypeFAQ}
	if err := h.requireManagedKBContentAccess(contributor, faq); err == nil {
		t.Fatal("managed FAQ mutations must be admin-only")
	}
	_, admin := managedBatchContext(t, types.TenantRoleAdmin, "admin")
	if err := h.requireManagedKBContentAccess(admin, faq); err != nil {
		t.Fatalf("managed admin should be allowed: %v", err)
	}
}
