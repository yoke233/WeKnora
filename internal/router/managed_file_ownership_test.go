package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func TestDocumentMutationPreservesLegacyKBOwnership(t *testing.T) {
	for _, test := range []struct {
		name    string
		tenant  uint64
		user    string
		creator *string
		kbType  string
		want    int
	}{
		{name: "legacy null document remains writable by KB creator", tenant: 7, user: "kb-owner", want: http.StatusNoContent},
		{name: "legacy uploader does not acquire KB ownership", tenant: 7, user: "uploader", creator: new("uploader"), want: http.StatusForbidden},
		{name: "managed document belongs to uploader", tenant: 42, user: "uploader", creator: new("uploader"), kbType: types.KnowledgeBaseTypeDocument, want: http.StatusNoContent},
		{name: "managed FAQ cannot use document owner bypass", tenant: 42, user: "uploader", creator: new("uploader"), kbType: types.KnowledgeBaseTypeFAQ, want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			enabled := true
			knowledge := &types.Knowledge{ID: "document", KnowledgeBaseID: "kb", TenantID: test.tenant, CreatorID: test.creator}
			kb := &types.KnowledgeBase{ID: "kb", TenantID: test.tenant, CreatorID: "kb-owner", Type: test.kbType}
			guards := &rbacGuards{
				cfg:              &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enabled, FileOwnershipTenantID: 42}},
				knowledgeService: &downloadKnowledgeLookup{knowledge: knowledge},
				kbService:        &stubWikiKBLookup{kbs: map[string]*types.KnowledgeBase{"kb": kb}},
			}
			r := gin.New()
			r.Use(func(c *gin.Context) {
				ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, test.tenant)
				ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleContributor)
				ctx = context.WithValue(ctx, types.UserIDContextKey, test.user)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
			})
			r.PUT("/knowledge/:id", guards.managedKnowledgePolicy("id"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/knowledge/document", nil))
			if recorder.Code != test.want {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, test.want, recorder.Body.String())
			}
		})
	}
}
