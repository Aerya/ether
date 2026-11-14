package transcode

import (
	"context"
	"testing"
	"time"

	"github.com/braedonsaunders/ether/internal/config"
	"github.com/braedonsaunders/ether/internal/core"
)

func TestQualityLadderInvariants(t *testing.T) {
	fr := newFakeRunner()
	m := newTestManager(t, fr, config.Transcode{}, WithSegmentTimeout(time.Second), withRungDwell(0))
	req := remuxRequest(t, "lazy-quality", time.Minute)
	req.Info = ladderInfo(1080, 15000)
	req.Decision = core.Decision{Kind: core.DecisionRemux, VideoCopy: true, AudioCopy: true, AudioStream: 1, SubtitleStream: -1, Container: ContainerFMP4}
	rungs := QualityRungs(req.Info, req.Decision, []string{"h264"}, 1280, 720, 2000)
	if len(rungs) == 0 {
		t.Fatal("no lower quality")
	}
	for _, r := range rungs {
		if r.Width > 1280 || r.Height > 720 || r.Bandwidth > 2000000 {
			t.Fatal("cap exceeded", r)
		}
	}
	if got := QualityRungs(req.Info, req.Decision, nil, 0, 0, 0); len(got) != 0 {
		t.Fatal("undeclared decoder", got)
	}
	req.QualityRungs, req.LazyStart, req.Continuous = true, true, true
	req.LadderVideoCodecs = []string{"h264"}
	for _, r := range rungs {
		req.QualityRungIDs = append(req.QualityRungIDs, r.ID)
	}
	s := startSession(t, m, req)
	if fr.startCount() != 0 || len(s.Rungs()) != len(rungs)+1 {
		t.Fatal("planning started work or lost ladder", fr.startCount(), s.Rungs())
	}
	if err := s.PrepareRung(context.Background(), rungs[0].ID, 0); err != nil {
		t.Fatal(err)
	}
	if fr.startCount() != 1 {
		t.Fatal("preparation did not select exactly one encoder", fr.startCount())
	}
	if err := s.PrepareRung(context.Background(), rungs[0].ID, 0); err != nil {
		t.Fatal(err)
	}
	if fr.startCount() != 1 {
		t.Fatal("prepared media restarted encoder")
	}
}
