package scheduling

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateVolunteerRequest_UserIDNullability(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantSet   bool
		wantValue *int
	}{
		{
			name:      "user_id omitted",
			body:      `{"nickname":"abc"}`,
			wantSet:   false,
			wantValue: nil,
		},
		{
			name:      "user_id explicit null unlinks user",
			body:      `{"user_id":null}`,
			wantSet:   true,
			wantValue: nil,
		},
		{
			name:      "user_id set to value",
			body:      `{"user_id":42}`,
			wantSet:   true,
			wantValue: intPtr(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req UpdateVolunteerRequest
			err := json.Unmarshal([]byte(tt.body), &req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSet, req.UserID.Set)
			assert.Equal(t, tt.wantValue, req.UserID.Value)
		})
	}
}

func intPtr(v int) *int { return &v }
