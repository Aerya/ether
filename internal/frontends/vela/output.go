package vela

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/braedonsaunders/ether/internal/frontend"
)

// refreshOutput reads only existing sessions. Codec claims come from real init
// bytes; planned and unused rungs never start work. Caller holds s.mu.
func (i *instance) refreshOutput(ctx context.Context, id string, s *playSession) bool {
	hs, ok := i.host.Sessions.TranscodeFor(ctx, id)
	if !ok || hs == nil {
		return false
	}
	paths, _ := s.plan["paths"].([]object)
	before, _ := json.Marshal(object{"paths": paths, "preparation": s.plan["preparation"]})
	for _, path := range paths {
		url, _ := path["url"].(string)
		if path["transport"] != "hls" || url == "" {
			continue
		}
		if ladder, ok := hs.(interface{ Rungs() []frontend.HLSRung }); ok {
			qualities := []object{}
			delete(path, "initialRenditionId")
			base := hlsDirectory(url)
			for _, r := range ladder.Rungs() {
				if r.Width <= 0 || r.Height <= 0 || r.Bandwidth <= 0 || r.ID == "" || strings.ContainsAny(r.ID, "/\\?#%\r\n") {
					continue
				}
				q := object{"id": r.ID, "width": r.Width, "height": r.Height, "bitrate": r.Bandwidth, "url": base + r.ID + "/index.m3u8", "videoCodec": "", "codecStatus": "pending"}
				if r.OutputCodecs != "" {
					codecs := strings.Split(r.OutputCodecs, ",")
					q["videoCodec"], q["codecStatus"] = codecs[0], "verified"
					if len(codecs) > 1 {
						q["audioCodec"] = codecs[1]
						if path["id"] == s.plan["selectedPathId"] {
							tracks, _ := s.plan["audioTracks"].([]object)
							for _, track := range tracks {
								if track["id"] == s.plan["selectedAudioTrackId"] && r.OutputAudioChannels > 0 {
									track["channels"] = r.OutputAudioChannels
								}
							}
						}
					}
				}
				initial := s.rungFor(path)
				if initial == "" {
					initial = "base"
				}
				if r.ID == initial {
					path["initialRenditionId"] = r.ID
				}
				qualities = append(qualities, q)
			}
			path["qualities"] = qualities
		}
	}
	if diagnostics, ok := hs.(frontend.HLSDiagnosticSession); ok {
		d := diagnostics.PlaybackDiagnostics()
		s.diagnostics = d.Records
		events := append([]frontend.HLSPreparationEvent{}, s.preparation...)
		events = append(events, d.Events...)
		s.plan["preparation"] = object{"sourceGeneration": s.source, "seekGeneration": s.seek, "events": events}
	}
	after, _ := json.Marshal(object{"paths": paths, "preparation": s.plan["preparation"]})
	return string(before) != string(after)
}
