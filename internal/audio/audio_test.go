package audio

import "testing"

func TestParseVolumeSettings(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantOutput   int
		wantInput    int
		wantMuted    bool
	}{
		{
			name:       "normal volume",
			input:      "output volume:50, input volume:75, alert volume:100, output muted:false",
			wantOutput: 50,
			wantInput:  75,
			wantMuted:  false,
		},
		{
			name:       "muted",
			input:      "output volume:0, input volume:100, alert volume:100, output muted:true",
			wantOutput: 0,
			wantInput:  100,
			wantMuted:  true,
		},
		{
			name:       "max volume",
			input:      "output volume:100, input volume:50, alert volume:100, output muted:false",
			wantOutput: 100,
			wantInput:  50,
			wantMuted:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseVolumeSettings(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.OutputVolume != tt.wantOutput {
				t.Errorf("OutputVolume = %d, want %d", info.OutputVolume, tt.wantOutput)
			}
			if info.InputVolume != tt.wantInput {
				t.Errorf("InputVolume = %d, want %d", info.InputVolume, tt.wantInput)
			}
			if info.Muted != tt.wantMuted {
				t.Errorf("Muted = %v, want %v", info.Muted, tt.wantMuted)
			}
		})
	}
}

func TestParseAudioJSON(t *testing.T) {
	input := []byte(`{
		"SPAudioDataType": [
			{
				"_name": "Audio",
				"_items": [
					{
						"_name": "MacBook Pro Speakers",
						"coreaudio_default_audio_output_device": "spaudio_yes",
						"coreaudio_output_source": "Internal Speakers"
					},
					{
						"_name": "MacBook Pro Microphone",
						"coreaudio_default_audio_input_device": "spaudio_yes",
						"coreaudio_input_source": "Internal Microphone"
					}
				]
			}
		]
	}`)

	devices, err := parseAudioJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	if devices[0].Name != "MacBook Pro Speakers" {
		t.Errorf("first device Name = %q, want %q", devices[0].Name, "MacBook Pro Speakers")
	}
	if devices[0].Type != "output" {
		t.Errorf("first device Type = %q, want %q", devices[0].Type, "output")
	}
	if !devices[0].Active {
		t.Error("expected first device to be active")
	}

	if devices[1].Name != "MacBook Pro Microphone" {
		t.Errorf("second device Name = %q, want %q", devices[1].Name, "MacBook Pro Microphone")
	}
	if devices[1].Type != "input" {
		t.Errorf("second device Type = %q, want %q", devices[1].Type, "input")
	}
	if !devices[1].Active {
		t.Error("expected second device to be active")
	}
}
