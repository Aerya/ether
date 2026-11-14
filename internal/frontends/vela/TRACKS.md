Vela playback uses the shared broker and lazy transcode ladder. A ready HLS path
publishes scoped rendition URLs, dimensions and peak bitrates. codecStatus starts
pending with empty videoCodec; actual fMP4 initialization bytes supply RFC6381
video/audio codec tokens and promote that rung to verified in a newer plan/SSE
revision of the same generation. Merely opening a session or reading playlists
starts no encoder. Requesting a rendition's init/segment selects that rung.
The path URL is the ABR master; qualities[].url allows manual rendition selection.
The on-demand original-file fallback has no URL or prepared rungs before switch.

Track.delivery follows Vela protocol/v0: external (webvtt or webvtt-window plus
URL), embedded, or burn-in. Track IDs/streamIndex are actual full source stream
indices and remain stable across audio, subtitle and quality changes. PGS,
VobSub and DVB retain their image codec. They are never advertised as WebVTT.
Image subtitle selection and off use tracks commands, restart output at the last
accepted observed title position, and advance source and seek generations.
Audio selection follows that same command path. The source revision stays pinned.

WebVTT windows reuse the playback demux cue tee; no separate source scan or
subtitle encoder is launched. delivery.url and legacy track.url carry the scoped
media grant. Set from_ms to ABSOLUTE title milliseconds. Each response covers
[from_ms, from_ms + 120000); returned cue timestamps are ABSOLUTE title seconds,
including after seek. Convert HLS-relative media time at the player clock boundary
using path.timelineOriginSeconds. Empty windows mean cues have not been published
or there are no cues in that interval: retry while playback is interested (about
one second is sufficient), cancel requests on generation changes, and fetch the
next window before its end. Do not interpret empty windows as subtitle failure.
