package core

import (
	"fmt"
	"testing"

	"github.com/synology-community/go-synology/pkg/api"
)

func TestExpandNotificationTemplateSettings_SortedOrder(t *testing.T) {
	input := map[string]bool{
		"z_tag": false,
		"a_tag": true,
		"m_tag": true,
	}

	got := expandNotificationTemplateSettings(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 settings, got %d", len(got))
	}

	if got[0].Tag != "a_tag" || !got[0].Enabled {
		t.Fatalf("unexpected first setting: %#v", got[0])
	}
	if got[1].Tag != "m_tag" || !got[1].Enabled {
		t.Fatalf("unexpected second setting: %#v", got[1])
	}
	if got[2].Tag != "z_tag" || got[2].Enabled {
		t.Fatalf("unexpected third setting: %#v", got[2])
	}
}

func TestFlattenNotificationTemplateSettings(t *testing.T) {
	got := flattenNotificationTemplateSettings(nil)
	if got.IsNull() || got.IsUnknown() {
		t.Fatalf("expected concrete map value")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "typed not found error",
			err:  api.NotFoundError(api.ApiError{Code: 404}),
			want: true,
		},
		{
			name: "raw api error with 404",
			err:  api.ApiError{Code: 404},
			want: true,
		},
		{
			name: "wrapped not found error",
			err:  fmt.Errorf("wrapped: %w", api.NotFoundError(api.ApiError{Code: 404})),
			want: true,
		},
		{
			name: "non-404 api error",
			err:  api.ApiError{Code: 105},
			want: false,
		},
		{
			name: "generic error",
			err:  fmt.Errorf("network timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundError(tt.err); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
