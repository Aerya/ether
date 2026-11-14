package stream

import (
	"github.com/braedonsaunders/ether/internal/archive"
	"github.com/braedonsaunders/ether/internal/nzb"
	"testing"
)

func TestMoviePackMembers(t *testing.T) {
	doc, err := nzb.Parse([]byte(`<?xml version="1.0"?><nzb xmlns="http://www.newzbin.com/DTD/2003/nzb"><head><meta type="ether-movie">{"Titles":["The Mummy"],"Year":1999}</meta></head><file subject="&quot;The Mummy.rar&quot; yEnc" date="1" poster="test"><groups><group>test</group></groups><segments><segment bytes="100" number="1">test</segment></segments></file></nzb>`))
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{rel: &release{doc: doc}}
	members := []archive.Member{{Name: "The Mummy Returns.mkv", Size: 9000}, {Name: "The Mummy (1959).mkv", Size: 8000}, {Name: "The Mummy (1999).mkv", Size: 5000}}
	if got := b.chooseMember(members); got == nil || got.Name != members[2].Name {
		t.Fatalf("selected %+v", got)
	}
	if got := b.chooseMember(members[:2]); got != nil {
		t.Fatalf("wrong collection member %+v", got)
	}
}
