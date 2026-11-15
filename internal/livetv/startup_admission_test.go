package livetv

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/braedonsaunders/ether/internal/core"
)

type admissionTranscoder struct{ budgetTranscoder }

func (admissionTranscoder) ProbeLive(context.Context, string) (*core.MediaInfo, error) {
	return &core.MediaInfo{Container: "mpegts", Streams: []core.StreamInfo{{Type: "video", Codec: "h264", Width: 1280, Height: 720}, {Type: "audio", Codec: "aac", Channels: 2, SampleRate: 48000}}}, nil
}

// Admission must exhaust failed inputs before returning a typed failure, while
// a working backup remains playable on the same tune and provider reservation.
func TestLiveStartupAdmission(t *testing.T) {
	for _, backupWorks := range []bool{false, true} {
		name := "all_refused"
		if backupWorks {
			name = "backup_plays"
		}
		t.Run(name, func(t *testing.T) {
			var attempted []string
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempted = append(attempted, r.URL.Path)
				if backupWorks && r.URL.Path == "/backup" {
					w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
					w.Write([]byte("#EXTM3U\n#EXT-X-TARGETDURATION:2\n#EXTINF:2,\nseg.ts\n"))
					return
				}
				w.WriteHeader(502)
			}))
			defer up.Close()
			var logs diagnosticBuffer
			s := testService(t, Options{Transcoder: admissionTranscoder{}, HTTPClient: up.Client(), Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
			c, err := s.SaveChannel(t.Context(), Channel{Name: "Startup", Number: "1", Kind: "iptv", Enabled: true, StreamSources: []StreamSource{{ID: "primary", URL: up.URL + "/primary", Enabled: true}, {ID: "backup", URL: up.URL + "/backup", Enabled: true, Priority: 1}}})
			if err != nil {
				t.Fatal(err)
			}
			started := time.Now()
			var admitted *steadyAdmission
			var code string
			if backupWorks {
				admitted, code, _, err = s.admitSteady(t.Context(), c)
			} else {
				_, err = s.Tune(t.Context(), &core.User{ID: 1}, c.ID, core.ClientCaps{})
			}
			t.Logf("admission elapsed=%s", time.Since(started))
			// The relay retains its bounded same-input retries before failover.
			var order []string
			for _, path := range attempted {
				if len(order) == 0 || path != order[len(order)-1] {
					order = append(order, path)
				}
			}
			if len(order) != 2 || order[0] != "/primary" || order[1] != "/backup" {
				t.Fatalf("input order: %v", attempted)
			}
			if backupWorks {
				if err != nil || code != "" || admitted == nil || admitted.link.src.ID != "backup" {
					t.Fatalf("backup lost: %v %s", err, code)
				}
				admitted.link.close(s)
			} else {
				if strings.Count(logs.text(), `"msg":"live tune failed"`) != 1 || !strings.Contains(logs.text(), `"level":"INFO"`) || len(s.ViewingSessions()) != 0 {
					t.Fatal("failed tune not summarized or released")
				}
				var detail *TuneError
				if !errors.As(err, &detail) || detail.UpstreamStatus != 502 || len(detail.Inputs) != 2 || !strings.Contains(detail.Error(), "HTTP 502") || admitted != nil || code != "" {
					t.Fatalf("failure evidence: %#v %s", err, code)
				}
			}
			s.mu.Lock()
			for _, reservations := range s.broadcastProviders {
				if len(reservations) != 0 {
					t.Error("provider reservation leaked")
				}
			}
			s.mu.Unlock()
		})
	}
}
