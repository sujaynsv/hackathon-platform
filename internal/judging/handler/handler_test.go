package handler_test

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/dogfood-platform/dogfood/internal/judging/domain"
    "github.com/dogfood-platform/dogfood/internal/judging/handler"
    "github.com/dogfood-platform/dogfood/internal/judging/port"
)

type mockQueueUseCase struct {
    getQueue func(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) (*port.AssignmentListDTO, error)
}
func (m *mockQueueUseCase) GetQueue(ctx context.Context, judgeID uuid.UUID, q port.QueueQuery) (*port.AssignmentListDTO, error) {
    if m.getQueue != nil {
        return m.getQueue(ctx, judgeID, q)
    }
    return &port.AssignmentListDTO{}, nil
}

type mockDetailUseCase struct {
    getDetail func(ctx context.Context, judgeID, assignmentID uuid.UUID) (*port.AssignmentDetailDTO, error)
}
func (m *mockDetailUseCase) GetDetail(ctx context.Context, judgeID, assignmentID uuid.UUID) (*port.AssignmentDetailDTO, error) {
    if m.getDetail != nil {
        return m.getDetail(ctx, judgeID, assignmentID)
    }
    return &port.AssignmentDetailDTO{}, nil
}

func setupRouter(h *handler.JudgingHandler, userID string) *chi.Mux {
    r := chi.NewRouter()
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), "user_id", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    })
    h.RegisterRoutes(r)
    return r
}

func TestJudgingHandler_GetQueue(t *testing.T) {
    judgeID := uuid.New()
    mockQueue := &mockQueueUseCase{
        getQueue: func(ctx context.Context, jid uuid.UUID, q port.QueueQuery) (*port.AssignmentListDTO, error) {
            assert.Equal(t, judgeID, jid)
            assert.Equal(t, "pending", q.Status)
            return &port.AssignmentListDTO{
                Data: []port.QueueRow{
                    {AssignmentID: uuid.New(), Status: domain.AssignmentPending},
                },
                Meta: port.ListMeta{TotalCount: 1},
            }, nil
        },
    }

    h := handler.NewJudgingHandler(mockQueue, nil, nil)
    r := setupRouter(h, judgeID.String())

    req := httptest.NewRequest(http.MethodGet, "/judging/queue", nil)
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    
    var resp struct {
        Data []port.QueueRow `json:"data"`
        Meta port.ListMeta   `json:"meta"`
    }
    require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
    assert.Len(t, resp.Data, 1)
    assert.Equal(t, 1, resp.Meta.TotalCount)
}

func TestJudgingHandler_GetAssignmentDetail(t *testing.T) {
    judgeID := uuid.New()
    assignmentID := uuid.New()
    
    mockDetail := &mockDetailUseCase{
        getDetail: func(ctx context.Context, jid, aid uuid.UUID) (*port.AssignmentDetailDTO, error) {
            assert.Equal(t, judgeID, jid)
            assert.Equal(t, assignmentID, aid)
            return &port.AssignmentDetailDTO{
                ID: aid,
                Status: "in_progress",
            }, nil
        },
    }

    h := handler.NewJudgingHandler(nil, mockDetail, nil)
    r := setupRouter(h, judgeID.String())

    req := httptest.NewRequest(http.MethodGet, "/judging/assignments/"+assignmentID.String(), nil)
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    
    var resp struct {
        Data port.AssignmentDetailDTO `json:"data"`
    }
    require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
    assert.Equal(t, assignmentID, resp.Data.ID)
    assert.Equal(t, "in_progress", resp.Data.Status)
}

type mockSubmitScoresUseCase struct {
    submitScores func(ctx context.Context, cmd port.SubmitScoresCommand) (*port.ScoreResultDTO, error)
}
func (m *mockSubmitScoresUseCase) SubmitScores(ctx context.Context, cmd port.SubmitScoresCommand) (*port.ScoreResultDTO, error) {
    if m.submitScores != nil {
        return m.submitScores(ctx, cmd)
    }
    return &port.ScoreResultDTO{}, nil
}

func TestJudgingHandler_SubmitScores(t *testing.T) {
    judgeID := uuid.New()
    assignmentID := uuid.New()
    critID := uuid.New()
    
    mockSubmit := &mockSubmitScoresUseCase{
        submitScores: func(ctx context.Context, cmd port.SubmitScoresCommand) (*port.ScoreResultDTO, error) {
            assert.Equal(t, judgeID, cmd.JudgeID)
            assert.Equal(t, assignmentID, cmd.AssignmentID)
            assert.Len(t, cmd.Scores, 1)
            assert.Equal(t, critID, cmd.Scores[0].CriterionID)
            assert.Equal(t, 8, cmd.Scores[0].RawScore)
            
            return &port.ScoreResultDTO{
                AssignmentID: assignmentID.String(),
                Status: "completed",
            }, nil
        },
    }

    h := handler.NewJudgingHandler(nil, nil, mockSubmit)
    r := setupRouter(h, judgeID.String())

    body := `{"scores": [{"criterionId": "` + critID.String() + `", "rawScore": 8}]}`
    req := httptest.NewRequest(http.MethodPost, "/judging/assignments/"+assignmentID.String()+"/scores", bytes.NewBufferString(body))
    req.Header.Set("Content-Type", "application/json")
    
    rr := httptest.NewRecorder()
    r.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusOK, rr.Code)
    
    var resp struct {
        Data port.ScoreResultDTO `json:"data"`
    }
    require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
    assert.Equal(t, assignmentID.String(), resp.Data.AssignmentID)
    assert.Equal(t, "completed", resp.Data.Status)
}
