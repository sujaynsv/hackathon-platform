package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOK(w *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	type stubData struct {
		Message string `json:"message"`
	}

	data := stubData{Message: "hello"}
	OK(rr, req, data)

	assert.Equal(w, http.StatusOK, rr.Code)

	var res ApiResponse[stubData]
	err := json.Unmarshal(rr.Body.Bytes(), &res)
	assert.NoError(w, err)
	assert.NotNil(w, res.Data)
	assert.Equal(w, "hello", res.Data.Message)
	assert.NotEmpty(w, res.Meta.RequestID)
	assert.NotEmpty(w, res.Meta.Timestamp)
}
