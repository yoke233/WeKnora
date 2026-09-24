package session

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubSearchSessionService struct {
	interfaces.SessionService
	calls   int
	weights types.RRFWeightOverride
}

func (s *stubSearchSessionService) SearchKnowledge(
	_ context.Context, _ []string, _ []string, _ []types.TagScope, _ string, weights types.RRFWeightOverride,
) ([]*types.SearchResult, error) {
	s.calls++
	s.weights = weights
	return []*types.SearchResult{{
		Content:   "chunk ![c](" + testResourceHandle + ")",
		ImageInfo: `[{"url":"` + testResourceHandle + `"}]`,
	}}, nil
}

func TestSearchKnowledgePassesRequestRRFWeights(t *testing.T) {
	stub := &stubSearchSessionService{}
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/knowledge-search", (&Handler{sessionService: stub}).SearchKnowledge)
	request := httptest.NewRequest(http.MethodPost, "/knowledge-search",
		bytes.NewBufferString(`{"query":"student id","knowledge_base_id":"kb-1","rrf_vector_weight":0.3,"rrf_keyword_weight":0.7}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusOK || stub.calls != 1 || stub.weights.RRFVectorWeight == nil ||
		stub.weights.RRFKeywordWeight == nil ||
		*stub.weights.RRFVectorWeight != 0.3 || *stub.weights.RRFKeywordWeight != 0.7 {
		t.Fatalf("request weights not forwarded: status=%d calls=%d weights=%+v body=%s",
			response.Code, stub.calls, stub.weights, response.Body.String())
	}
}

func TestSearchKnowledgeRejectsIncompleteRRFWeights(t *testing.T) {
	stub := &stubSearchSessionService{}
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/knowledge-search", (&Handler{sessionService: stub}).SearchKnowledge)
	request := httptest.NewRequest(http.MethodPost, "/knowledge-search",
		bytes.NewBufferString(`{"query":"student id","knowledge_base_id":"kb-1","rrf_keyword_weight":0.7}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || stub.calls != 0 {
		t.Fatalf("incomplete weights reached search: status=%d calls=%d body=%s",
			response.Code, stub.calls, response.Body.String())
	}
}

func TestSearchKnowledge_PublicResourceURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	h := &Handler{
		sessionService: &stubSearchSessionService{},
		fileService:    &stubResourceFileService{},
	}
	r.POST("/knowledge-search", h.SearchKnowledge)

	body := bytes.NewBufferString(`{"query":"diagram","knowledge_base_ids":["kb-1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/knowledge-search?resource_urls=public", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), testResourceHandle)
	assert.Contains(t, w.Body.String(), "cdn.example.com")
}

func TestSearchKnowledge_InvalidResourceURLMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	h := &Handler{
		sessionService: &stubSearchSessionService{},
		fileService:    &stubResourceFileService{},
	}
	r.POST("/knowledge-search", h.SearchKnowledge)

	body := bytes.NewBufferString(`{"query":"diagram","knowledge_base_ids":["kb-1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/knowledge-search?resource_urls=signed", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "resource_urls")
}

func TestSearchKnowledge_DefaultKeepsHandles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	h := &Handler{
		sessionService: &stubSearchSessionService{},
		fileService:    &stubResourceFileService{},
	}
	r.POST("/knowledge-search", h.SearchKnowledge)

	body := bytes.NewBufferString(`{"query":"diagram","knowledge_base_ids":["kb-1"]}`)
	req := httptest.NewRequest(http.MethodPost, "/knowledge-search", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

	var resp struct {
		Data []struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Contains(t, resp.Data[0].Content, testResourceHandle)
}
