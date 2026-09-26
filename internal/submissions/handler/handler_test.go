package handler_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/dogfood-platform/dogfood/internal/submissions/handler"
	"github.com/dogfood-platform/dogfood/internal/submissions/port"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUploadSvc struct {
	mock.Mock
}

func (m *mockUploadSvc) Upload(ctx context.Context, cmd port.UploadCommand) (*port.UploadDTO, error) {
	args := m.Called(ctx, cmd)
	var res *port.UploadDTO
	if args.Get(0) != nil {
		res = args.Get(0).(*port.UploadDTO)
	}
	return res, args.Error(1)
}

type mockListSvc struct {
	mock.Mock
}

func (m *mockListSvc) ListFiles(ctx context.Context, submissionID uuid.UUID) ([]*port.UploadDTO, error) {
	args := m.Called(ctx, submissionID)
	var res []*port.UploadDTO
	if args.Get(0) != nil {
		res = args.Get(0).([]*port.UploadDTO)
	}
	return res, args.Error(1)
}

func setupRouter(upload *mockUploadSvc, list *mockListSvc, userID uuid.UUID) *chi.Mux {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "user_id", userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	h := handler.NewSubmissionHandler(nil, nil, nil, upload, list, nil, nil)
	h.RegisterRoutes(r)
	return r
}

func TestUploadHandler_ValidJPEG_Returns201(t *testing.T) {
	uploadSvc := new(mockUploadSvc)
	listSvc := new(mockListSvc)
	userID := uuid.New()
	r := setupRouter(uploadSvc, listSvc, userID)

	subID := uuid.New()

	var b bytes.Buffer
	wForm := multipart.NewWriter(&b)
	part, _ := wForm.CreateFormFile("file", "test.jpg")
	part.Write([]byte("fake image data"))
	wForm.WriteField("type", "cover")
	wForm.Close()

	req := httptest.NewRequest(http.MethodPost, "/submissions/"+subID.String()+"/upload", &b)
	req.Header.Set("Content-Type", wForm.FormDataContentType())
	w := httptest.NewRecorder()

	uploadSvc.On("Upload", mock.Anything, mock.Anything).Return(&port.UploadDTO{
		ID:           uuid.New(),
		SubmissionID: subID,
		URL:          "https://minio/uploads-covers/...",
		ContentType:  "application/octet-stream",
		SizeBytes:    15,
		Type:         "cover",
		UploadedAt:   time.Now().UTC(),
	}, nil)

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestUploadHandler_TooLarge_Returns413(t *testing.T) {
	uploadSvc := new(mockUploadSvc)
	listSvc := new(mockListSvc)
	userID := uuid.New()
	r := setupRouter(uploadSvc, listSvc, userID)

	subID := uuid.New()

	var b bytes.Buffer
	wForm := multipart.NewWriter(&b)
	wForm.WriteField("type", "cover")
	
	part, _ := wForm.CreateFormFile("file", "test.jpg")
	// Write 51MB
	largeData := strings.Repeat("a", 51*1024*1024)
	part.Write([]byte(largeData))
	wForm.Close()

	req := httptest.NewRequest(http.MethodPost, "/submissions/"+subID.String()+"/upload", &b)
	req.Header.Set("Content-Type", wForm.FormDataContentType())
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestUploadHandler_InvalidMIME_Returns400(t *testing.T) {
	uploadSvc := new(mockUploadSvc)
	listSvc := new(mockListSvc)
	userID := uuid.New()
	r := setupRouter(uploadSvc, listSvc, userID)

	subID := uuid.New()

	var b bytes.Buffer
	wForm := multipart.NewWriter(&b)
	part, _ := wForm.CreateFormFile("file", "test.exe")
	part.Write([]byte("fake executable data"))
	wForm.WriteField("type", "cover")
	wForm.Close()

	req := httptest.NewRequest(http.MethodPost, "/submissions/"+subID.String()+"/upload", &b)
	req.Header.Set("Content-Type", wForm.FormDataContentType())
	w := httptest.NewRecorder()

	// In the handler, the MIME validation happens in domain.ValidateUpload, 
	// which is called by the usecase. The handler passes it down.
	// We need to return the error from the mock usecase.
	// Since handler expects DomainError, we can return response.ErrInvariantViolated for invalid MIME? 
	// Wait, we decided to return 422 for domain violations from usecase, but the spec says 400 INVALID_FILE_TYPE.
	// If the spec really wants 400 INVALID_FILE_TYPE from the handler, maybe I should map it in the handler or return a specific error from usecase that maps to 400.
	// Let's just mock what happens.
	uploadSvc.On("Upload", mock.Anything, mock.Anything).Return((*port.UploadDTO)(nil), response.ErrInvalidFileType)

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
