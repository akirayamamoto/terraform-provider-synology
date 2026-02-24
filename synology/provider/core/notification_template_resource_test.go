package core

import "testing"

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
