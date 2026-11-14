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

Original-file embedded delivery is offered only for explicitly declared source
subtitle codecs. Unsupported original text is omitted while its only possible
external URL would need an inactive HLS cue tee; it appears as external WebVTT
when fallback activates. Burn-in always means actual image subtitle burn-in.

File quality choices use the same broker ladder for direct-file, remux and
transcode starts. `primary` is the starting route (the menu's Original choice);
only actual original-file delivery carries `original:true`. Lower paths are
`quality-<rung ID>` with optional `quality:{width,height,bitrate}` target metadata
(bits/second). They are URL-less/on-demand until selected. The canonical Vela
protocol addition is d33421a. No label or decoder claim should be inferred from
an ID. The playback lead and protocol owner agreed these fields directly.

`switch` at absolute title seconds preserves position and advances both source
and seek generations. Actual HLS qualities remain real broker rendition URLs,
with pending/empty codecs until init bytes prove them. A prepared switch retains
the prepared timeline origin. Seek/track restarts invalidate prepared paths.
Original-file URLs stay ready and range-authorized while HLS is prepared/played.

`prepare-fallback` with positive maxBitrate selects the highest allowed lower rung
whose peak bitrate fits the limit, requests a bounded media range on the existing
ladder, and reports ready only after publication. It leaves the selected path and
generations unchanged. The host advertises proactiveFallback only through the
optional broker quality service backed by the local manager; unsupported hosts
keep it false. Capacity is still checked on actual output activation. Explicit
quality sessions retire the old run before starting a replacement, so the ladder
has one encoder at a time. Other surfaces retain their existing behavior.

Coordinator lab checks (no deployment/streaming performed by this worker):
- 1917 primary H264-copy/AAC remux: menu shows Original first plus lower choices;
  opening the session and fetching metadata starts no unused encoder.
- Choose 720p at a nonzero position, then Original; verify position/audio/subtitle
  selection and source/seek generations, including subsequent seek/track changes.
- While Original plays, prepare-fallback at 2000000 bits/second; selected path and
  generations stay unchanged, fallback becomes ready only with a real segment;
  switching to its path reuses the prepared origin and output.
- Observe process/admission diagnostics: at most one active encoder, repeated
  prepare reuses cached output, stop cleans up. Check MSE and native HLS Auto.
- Repeat with original-file and transcode starts, allowTranscode=false (Original
  only), constrained decoder/size/bitrate caps, and exhausted encoder capacity.
