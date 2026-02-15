package display

import (
	"testing"
)

func TestParseDisplayJSON(t *testing.T) {
	input := []byte(`{
		"SPDisplaysDataType": [
			{
				"_name": "Apple M1 Pro",
				"sppci_vendor": "Apple",
				"spdisplays_ndrvs": [
					{
						"_name": "Built-in Retina Display",
						"_spdisplays_resolution": "3024 x 1964 @ 120 Hz",
						"spdisplays_refresh_rate": "120 Hz",
						"spdisplays_main": "spdisplays_yes"
					}
				]
			}
		]
	}`)

	displays, err := parseDisplayJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(displays) != 1 {
		t.Fatalf("expected 1 display, got %d", len(displays))
	}

	d := displays[0]
	if d.Name != "Built-in Retina Display" {
		t.Errorf("Name = %q, want %q", d.Name, "Built-in Retina Display")
	}
	if d.Resolution != "3024 x 1964 @ 120 Hz" {
		t.Errorf("Resolution = %q, want %q", d.Resolution, "3024 x 1964 @ 120 Hz")
	}
	if d.Vendor != "Apple" {
		t.Errorf("Vendor = %q, want %q", d.Vendor, "Apple")
	}
	if !d.Main {
		t.Error("expected Main to be true")
	}
}

func TestParseDisplayJSONMultiple(t *testing.T) {
	input := []byte(`{
		"SPDisplaysDataType": [
			{
				"_name": "Apple M1 Pro",
				"sppci_vendor": "Apple",
				"spdisplays_ndrvs": [
					{
						"_name": "Built-in Retina Display",
						"_spdisplays_resolution": "3024 x 1964 @ 120 Hz",
						"spdisplays_refresh_rate": "120 Hz",
						"spdisplays_main": "spdisplays_yes"
					},
					{
						"_name": "LG UltraFine",
						"_spdisplays_resolution": "5120 x 2880 @ 60 Hz",
						"spdisplays_refresh_rate": "60 Hz",
						"spdisplays_main": "spdisplays_no"
					}
				]
			}
		]
	}`)

	displays, err := parseDisplayJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(displays) != 2 {
		t.Fatalf("expected 2 displays, got %d", len(displays))
	}

	if displays[1].Name != "LG UltraFine" {
		t.Errorf("second display Name = %q, want %q", displays[1].Name, "LG UltraFine")
	}
	if displays[1].Main {
		t.Error("expected second display Main to be false")
	}
}
