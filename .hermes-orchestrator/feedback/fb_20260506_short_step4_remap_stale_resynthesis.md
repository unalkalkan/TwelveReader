# Short-term Step 4: persona remap marks old audio stale and prioritizes new segments

Created: 2026-05-06T18:49:55Z
Source: Telegram / human project owner

When the user changes a persona mapping at any time:
- Newer/future segments for that persona should always be synthesized with the new voice first.
- Existing audio generated with the old persona voice should be marked stale, not immediately blocking new work.
- After the whole book is segmented and synthesized, the system can go back and regenerate stale segments.
- The UX should tolerate in-progress mapping changes and avoid blocking playback/progress unnecessarily.
