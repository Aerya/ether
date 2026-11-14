package sessions

import (
	"context"
	"time"

	"github.com/braedonsaunders/ether/internal/core"
	"github.com/braedonsaunders/ether/internal/frontend"
	"github.com/braedonsaunders/ether/internal/transcode"
)

func (b *Broker) ProactiveFallbackAvailable() bool {
	provider, ok := b.deps.Transcoder.(interface{ ProactiveFallbackAvailable() bool })
	return ok && provider.ProactiveFallbackAvailable()
}

func (b *Broker) qualityRungs(ctx context.Context, s *brokerSession, viewer *core.Viewer) ([]core.HLSRung, error) {
	if s.public.Protocol != "vela" || s.live || s.restored || !b.ProactiveFallbackAvailable() || !b.currentMediaPolicy(ctx, viewer, s) || (s.allowTranscoding != nil && !*s.allowTranscoding) {
		return nil, nil
	}
	ps, ok := b.deps.Playback.Session(s.playID)
	if !ok || ps == nil {
		return nil, core.ErrNotAvailable
	}
	if s.sourcePinned && sourceRevision(ps) != s.sourceETag {
		return nil, frontend.ErrSourceGone
	}
	dec := s.decision
	if dec.Reason == "original_runtime" {
		options := s.opts
		options.DirectPlay = false
		var err error
		dec, err = b.decideProfile(ps, frontend.PlanRequest{Viewer: viewer, Device: s.public.Device, Protocol: s.public.Protocol, Caps: s.caps, Intent: frontend.PlayIntent{Options: options, AllowTranscoding: s.allowTranscoding}})
		if err != nil {
			return nil, err
		}
	}
	dec, allowed := b.hlsOutputDecision(ps, s.caps, s.opts, s.allowTranscoding, dec)
	if !allowed || dec.Container == "mkv" {
		return nil, nil
	}
	width, height, kbps := s.opts.MaxWidth, s.opts.MaxHeight, s.opts.MaxBitrateKbps
	if s.caps.MaxHeight > 0 && (height == 0 || s.caps.MaxHeight < height) {
		height = s.caps.MaxHeight
	}
	if s.caps.MaxBitrateKbps > 0 && (kbps == 0 || s.caps.MaxBitrateKbps < kbps) {
		kbps = s.caps.MaxBitrateKbps
	}
	if b.deps.BitrateLimit != nil {
		if limit := b.deps.BitrateLimit(viewer); limit > 0 && (kbps == 0 || limit < kbps) {
			kbps = limit
		}
	}
	if s.caps.Remote && b.deps.RemoteBitrateLimit != nil {
		if limit := b.deps.RemoteBitrateLimit(); limit > 0 && (kbps == 0 || limit < kbps) {
			kbps = limit
		}
	}
	rungs := transcode.QualityRungs(ps.Info, dec, s.caps.VideoCodecs, width, height, kbps)
	out := rungs[:0]
	for _, rung := range rungs {
		d, ok := transcode.QualityDecision(ps.Info, dec, rung.ID)
		if limit := s.caps.VideoMaxHeights[d.VideoCodec]; limit > 0 && rung.Height > limit {
			continue
		}
		if ok && profileDeliveryAllowed(s.caps.Profiles, d, ps.Info, "hls") {
			out = append(out, rung)
		}
	}
	return out, nil
}

func (b *Broker) QualityRungs(ctx context.Context, a frontend.SessionAccess, id string) ([]core.HLSRung, error) {
	s, v, err := b.access(ctx, a, id)
	if err != nil {
		return nil, err
	}
	return b.qualityRungs(ctx, s, v)
}

func (b *Broker) PrepareQuality(ctx context.Context, a frontend.SessionAccess, id, rung string, position time.Duration) (string, time.Duration, error) {
	rungs, err := b.QualityRungs(ctx, a, id)
	if err != nil {
		return "", 0, err
	}
	allowed := false
	for _, r := range rungs {
		if r.ID == rung {
			allowed = true
		}
	}
	if !allowed {
		return "", 0, core.ErrNotAvailable
	}
	s, _, err := b.access(ctx, a, id)
	if err != nil {
		return "", 0, err
	}
	origin := position
	if hs, ok := b.transcodeSession(id); ok && hs != nil {
		origin = s.streamPosition
	}
	url, err := b.StartStream(ctx, a, id, frontend.StreamRequest{Format: frontend.StreamHLS, Position: origin, LazyStart: true, Continuous: true, PrepareOnly: true})
	if err != nil {
		return "", 0, err
	}
	hs, ok := b.transcodeSession(id)
	if !ok {
		return "", 0, core.ErrNotAvailable
	}
	preparer, ok := hs.(interface {
		PrepareRung(context.Context, string, time.Duration) error
	})
	if !ok {
		return "", 0, core.ErrNotAvailable
	}
	if err = preparer.PrepareRung(ctx, rung, position); err != nil {
		return "", 0, err
	}
	return url, origin, nil
}
