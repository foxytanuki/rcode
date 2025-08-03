package api

import (
	"testing"
	"time"
)

func TestOpenRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request OpenRequest
		wantErr error
	}{
		{
			name: "valid request",
			request: OpenRequest{
				Path: "/home/user/project",
				User: "testuser",
				Host: "remote.example.com",
			},
			wantErr: nil,
		},
		{
			name: "missing path",
			request: OpenRequest{
				Path: "",
				User: "testuser",
				Host: "remote.example.com",
			},
			wantErr: ErrInvalidPath,
		},
		{
			name: "missing user",
			request: OpenRequest{
				Path: "/home/user/project",
				User: "",
				Host: "remote.example.com",
			},
			wantErr: ErrMissingUser,
		},
		{
			name: "missing host",
			request: OpenRequest{
				Path: "/home/user/project",
				User: "testuser",
				Host: "",
			},
			wantErr: ErrMissingHost,
		},
		{
			name: "with optional editor",
			request: OpenRequest{
				Path:   "/home/user/project",
				User:   "testuser",
				Host:   "remote.example.com",
				Editor: "vscode",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if err != tt.wantErr {
				t.Errorf("OpenRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenRequest_SetTimestamp(t *testing.T) {
	req := &OpenRequest{
		Path: "/test",
		User: "user",
		Host: "host",
	}

	before := time.Now().Unix()
	req.SetTimestamp()
	after := time.Now().Unix()

	if req.Timestamp < before || req.Timestamp > after {
		t.Errorf("SetTimestamp() timestamp = %v, want between %v and %v", req.Timestamp, before, after)
	}
}

func TestOpenResponse_SetTimestamp(t *testing.T) {
	resp := &OpenResponse{
		Success: true,
		Message: "test",
	}

	before := time.Now().Unix()
	resp.SetTimestamp()
	after := time.Now().Unix()

	if resp.Timestamp < before || resp.Timestamp > after {
		t.Errorf("SetTimestamp() timestamp = %v, want between %v and %v", resp.Timestamp, before, after)
	}
}

func TestEditorsResponse_SetTimestamp(t *testing.T) {
	resp := &EditorsResponse{
		Editors: []EditorInfo{},
	}

	before := time.Now().Unix()
	resp.SetTimestamp()
	after := time.Now().Unix()

	if resp.Timestamp < before || resp.Timestamp > after {
		t.Errorf("SetTimestamp() timestamp = %v, want between %v and %v", resp.Timestamp, before, after)
	}
}

func TestHealthResponse_SetTimestamp(t *testing.T) {
	resp := &HealthResponse{
		Status: "healthy",
	}

	before := time.Now().Unix()
	resp.SetTimestamp()
	after := time.Now().Unix()

	if resp.Timestamp < before || resp.Timestamp > after {
		t.Errorf("SetTimestamp() timestamp = %v, want between %v and %v", resp.Timestamp, before, after)
	}
}

func TestHealthResponse_IsHealthy(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{
			name:   "healthy status",
			status: "healthy",
			want:   true,
		},
		{
			name:   "unhealthy status",
			status: "unhealthy",
			want:   false,
		},
		{
			name:   "empty status",
			status: "",
			want:   false,
		},
		{
			name:   "other status",
			status: "degraded",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &HealthResponse{
				Status: tt.status,
			}
			if got := resp.IsHealthy(); got != tt.want {
				t.Errorf("HealthResponse.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEditorInfo(t *testing.T) {
	editor := EditorInfo{
		Name:      "vscode",
		Command:   "code --remote ssh-remote+{user}@{host} {path}",
		Available: true,
		Default:   false,
	}

	if editor.Name != "vscode" {
		t.Errorf("EditorInfo.Name = %v, want %v", editor.Name, "vscode")
	}
	if !editor.Available {
		t.Errorf("EditorInfo.Available = %v, want %v", editor.Available, true)
	}
	if editor.Default {
		t.Errorf("EditorInfo.Default = %v, want %v", editor.Default, false)
	}
}
