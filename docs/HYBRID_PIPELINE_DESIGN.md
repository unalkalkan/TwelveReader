# Hybrid Pipeline Design - Parallel LLM→TTS→Playback with Incremental Voice Mapping

## Overview

This document describes the hybrid pipeline architecture for TwelveReader that enables parallel processing of book segmentation, TTS synthesis, and audio playback while handling voice mapping incrementally as new personas are discovered.

## Architecture Overview

### Flow Summary
1. **Segment first 5 segments** → Pause, discover initial personas
2. **User maps initial personas** → Start TTS + Playback pipelines
3. **Background segmentation continues** → Smart queuing based on voice mapping status
4. **New persona discovered** → Flag segments, show floating dialog, continue processing mapped segments
5. **User maps new persona** → Priority-queue waiting segments, resume playback when ready
6. **Repeat** until book complete

## Key Design Principles

1. **Zero Wait After Initial Mapping**: User can start listening within 30-60 seconds
2. **Non-Blocking Persona Discovery**: New character discovery doesn't stop the entire pipeline
3. **Smart Queuing**: Only segments with mapped personas proceed to TTS
4. **Seamless Playback**: Audio plays continuously until reaching an unmapped segment
5. **Priority Re-queueing**: Newly mapped personas get priority for immediate synthesis

---

## Backend Architecture

### 1. Enhanced Pipeline Orchestrator (`internal/pipeline/orchestrator.go`)

**Current State:** Basic parallel LLM→TTS with fixed threshold (5 segments)

**Required Changes:**

#### A. Persona Discovery Tracking
```go
type pipelineState struct {
    // ... existing fields ...
    
    // New fields for persona management
    discoveredPersonas    map[string]bool      // All personas seen so far
    mappedPersonas        map[string]string    // persona -> voiceID (from VoiceMap)
    unmappedPersonas      []string             // Personas waiting for mapping
    pendingSegments       []*types.Segment     // Segments with unmapped personas
    personaDiscoveryQueue chan PersonaDiscoveryEvent
}

type PersonaDiscoveryEvent struct {
    Personas    []string         // Newly discovered personas
    Segment     *types.Segment   // First segment with new persona
    TotalMapped int              // Total mapped personas so far
}
```

#### B. Smart Segment Queue
```go
// Replace simple channel with priority queue
type SegmentQueue struct {
    mappedQueue    []*types.Segment  // Ready for TTS
    unmappedQueue  []*types.Segment  // Waiting for voice mapping
    mu             sync.RWMutex
}

func (sq *SegmentQueue) Enqueue(segment *types.Segment, isMapped bool)
func (sq *SegmentQueue) DequeueNext() *types.Segment
func (sq *SegmentQueue) PromotePendingSegments(persona string)
```

#### C. Modified Segmentation Stage
- After **first 5 completed segments**, emit `VOICE_MAPPING_REQUIRED` event with discovered personas
- Continue segmentation but check each new segment for unmapped personas
- Route segments to appropriate queue (mapped vs unmapped)
- Emit `NEW_PERSONA_DISCOVERED` event when finding unmapped persona

#### D. Modified TTS Stage
- Only process segments from `mappedQueue`
- Skip segments with unmapped personas
- When new persona is mapped, promote their segments to front of queue
- Continue synthesis without blocking on unmapped segments

---

### 2. Voice Mapping State Machine

**New Status Flow:**
```
uploaded → parsing → segmenting_initial (0-5 segments)
    ↓
voice_mapping_required (pause at 5)
    ↓
voice_mapping_confirmed → synthesizing_and_segmenting (parallel)
    ↓
[If new persona] → new_persona_discovered (flag, don't pause pipeline)
    ↓
[User maps] → resume_synthesis
    ↓
synthesized (complete)
```

**New Book Fields:**
```go
type Book struct {
    // ... existing fields ...
    
    // New persona tracking
    DiscoveredPersonas  []string          `json:"discovered_personas"`
    UnmappedPersonas    []string          `json:"unmapped_personas"`
    PendingSegmentCount int               `json:"pending_segment_count"`
    WaitingForMapping   bool              `json:"waiting_for_mapping"`
}
```

---

### 3. New API Endpoints

#### `/api/v1/books/:id/pipeline/status` (GET)
Returns real-time pipeline status including:
- Stage progress (segmenting, synthesizing, ready)
- Discovered vs mapped personas
- Pending segments count
- Current playback-blocking status

**Response Format:**
```json
{
  "book_id": "book-123",
  "stages": [
    {
      "stage": "segmenting",
      "status": "in_progress",
      "current": 45,
      "total": 72,
      "percentage": 62.5,
      "message": "Analyzing book content",
      "started_at": "2026-01-30T10:00:00Z"
    },
    {
      "stage": "synthesizing",
      "status": "in_progress",
      "current": 17,
      "total": 45,
      "percentage": 37.8,
      "message": "17/45 segments (3 pending mapping)",
      "started_at": "2026-01-30T10:01:00Z"
    },
    {
      "stage": "ready",
      "status": "in_progress",
      "current": 17,
      "total": 45,
      "percentage": 37.8,
      "message": "17 segments available for playback"
    }
  ],
  "updated_at": "2026-01-30T10:05:00Z"
}
```

#### `/api/v1/books/:id/voice-map` (POST) - Enhanced
**Current:** Simple save + trigger full synthesis

**New Behavior:**
- **First call (initial 5 segments):** Start parallel pipeline
- **Subsequent calls (new personas):** Update mapping, promote pending segments, continue pipeline
- **Query param:** `?initial=true` for first mapping vs `?update=true` for new persona

**Request Format:**
```json
{
  "book_id": "book-123",
  "persons": [
    {
      "id": "narrator",
      "provider_voice": "voice-id-1"
    },
    {
      "id": "alice",
      "provider_voice": "voice-id-2"
    }
  ]
}
```

#### `/api/v1/books/:id/personas` (GET)
Returns discovered personas with mapping status:

**Response Format:**
```json
{
  "discovered": ["narrator", "alice", "bob"],
  "mapped": {
    "narrator": "voice-id-1",
    "alice": "voice-id-2"
  },
  "unmapped": ["bob"],
  "pending_segments": 3
}
```

---

## Frontend Architecture

### 1. Unified Progress View (Single Screen)

**Replace:** Tab-based navigation (Upload → Status → Voice Mapping → Player)

**New Layout:**
```
┌─────────────────────────────────────────────────────────┐
│  Book: "Alice in Wonderland"                            │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─ Segmenting ────────────────────────┐               │
│  │ ████████████████░░░░░░░░ 62%        │               │
│  │ 45/72 segments                      │               │
│  └─────────────────────────────────────┘               │
│                                                          │
│  ┌─ Synthesizing ──────────────────────┐               │
│  │ ████████████░░░░░░░░░░░░ 38%        │               │
│  │ 17/45 segments (3 pending mapping)  │               │
│  └─────────────────────────────────────┘               │
│                                                          │
│  ┌─ Ready for Playback ────────────────┐               │
│  │ ████████████░░░░░░░░░░░░ 38%        │               │
│  │ 17 segments available                │               │
│  └─────────────────────────────────────┘               │
│                                                          │
├─────────────────────────────────────────────────────────┤
│  Playback Controls                                       │
│  [◄] [▶] [►] Segment 12/17 ──●────────── 02:45         │
│                                                          │
│  Current Text:                                           │
│  "Alice was beginning to get very tired..."             │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 2. Floating Voice Mapping Dialog

**Trigger Conditions:**
- After first 5 segments completed (initial mapping)
- When new persona discovered during processing (update mapping)

**Design:**
```
┌──────────────────────────────────────┐
│  🎙️ Voice Mapping Required          │
├──────────────────────────────────────┤
│  New character discovered!           │
│                                      │
│  Person: "Bob"                       │
│  Description: Male, excited          │
│                                      │
│  Voice: [Select voice ▼]            │
│                                      │
│  Preview: [▶ Play sample]           │
│                                      │
│  [Cancel]  [Confirm Mapping]        │
│                                      │
│  ℹ️ 3 segments waiting for this      │
│     voice mapping                    │
└──────────────────────────────────────┘
```

**Positioning:** 
- Floating overlay on right side of playback UI
- Non-modal (doesn't block playback of existing audio)
- Persistent until user action
- Can show voice picker with preview

**Behavior:**
- **Initial Mapping (5 segments):** Shows all discovered personas at once
- **New Persona Discovery:** Shows one persona at a time, in order of appearance
- **Multiple Unmapped:** Queue dialogs, show sequentially
- **No Skip Option:** User must map the voice to continue

---

### 3. Enhanced BookPlayer Component

**Current:** Simple sequential playback, loads one segment at a time

**New Features:**

#### A. Smart Buffering
- Preload next 3 available segments
- Skip unmapped segments in playback queue
- Handle gaps gracefully

#### B. Playback Pause Logic
```typescript
// Pseudo-code
function getNextPlayableSegment(currentIndex: number): Segment | null {
  const nextSegment = segments[currentIndex + 1];
  
  if (!nextSegment) return null;
  
  if (nextSegment.hasAudio) {
    return nextSegment;
  }
  
  if (nextSegment.isUnmapped) {
    // Show "waiting for voice mapping" message
    // Keep checking for synthesis completion
    return null; // Pause here
  }
  
  if (nextSegment.isSynthesizing) {
    // Show "synthesizing..." message
    // Poll for completion
    return null; // Pause here
  }
}
```

#### C. Status Indicators
Show segment status in text display:
- ✅ Synthesized (green)
- 🎙️ Synthesizing (yellow spinner)
- ⏸️ Waiting for mapping (orange warning)
- ⏳ Queued for synthesis (gray)

#### D. Behavior Details

**When Playback Reaches Unmapped Segment:**
1. Playback pauses automatically
2. Show status: "⏸️ Waiting for voice mapping: Bob"
3. Floating dialog is already visible (was shown when persona discovered)
4. Continue polling for synthesis completion
5. Resume playback automatically when audio becomes available

**Segment Order in Playback:**
- Example: If [mapped] - [unmapped] - [mapped] are next 3 segments
- Segments 1 and 3 synthesize in background
- Playback continues through segment 1
- Playback pauses at segment 2 until mapped
- When mapped, segment 2 synthesizes with priority
- Playback resumes through segments 2 and 3

---

## Implementation Phases

### Phase 1: Backend Core (Priority 1)
1. **Modify Pipeline Orchestrator**
   - Add persona discovery tracking
   - Implement smart segment queue (mapped vs unmapped)
   - Add event system for persona discovery
   - Modify segmentation stage to pause at 5 segments
   - Modify TTS stage to skip unmapped segments

2. **Enhance Voice Mapping API**
   - Add initial vs update mapping logic
   - Implement segment re-queueing on new mapping
   - Add persona discovery endpoint

3. **Update Book Status Model**
   - Add persona tracking fields
   - Add pipeline status endpoint

### Phase 2: Frontend Core (Priority 2)
1. **Unified Progress View**
   - Single-page view with all three stages
   - Real-time WebSocket or polling for progress updates
   - Replace tab navigation

2. **Floating Voice Mapping Dialog**
   - Non-blocking modal on right side
   - Voice picker with preview
   - Shows pending segment count
   - Handles initial + update scenarios

3. **Enhanced Playback Logic**
   - Smart segment queue with gap handling
   - Pause at unmapped segments
   - Status indicators per segment
   - Resume when synthesis completes

### Phase 3: Polish (Priority 3)
1. **Error Handling**
   - TTS failures with retry
   - Network interruption recovery
   - Partial completion states

2. **UX Refinements**
   - Smooth transitions between states
   - Loading states and spinners
   - Audio preloading optimization
   - Keyboard shortcuts

3. **Analytics & Monitoring**
   - Track pipeline performance
   - Monitor synthesis costs
   - User flow analytics

---

## Key Technical Decisions

### 1. Segment Queue Priority System
**Selected: Option B - Dual Queues**

Use two separate queues:
- **mappedQueue**: Segments with all personas mapped, ready for TTS
- **unmappedQueue**: Segments waiting for voice mapping

**Rationale:** Simplest to implement and reason about. Clear separation of concerns.

### 2. Real-time Updates (Frontend ↔ Backend)
**Selected: Option C - Polling**

Poll API every 2 seconds during active processing.

**Rationale:** 
- Current codebase already uses polling for status updates
- Simple to implement and debug
- Sufficient for this use case (2s latency acceptable)
- Can upgrade to SSE/WebSocket later if needed

### 3. Segment Re-queueing Strategy
**Selected: Option A - Jump to Front**

When new persona is mapped, affected segments jump to front of TTS queue.

**Rationale:** 
- User is actively waiting for these segments to continue playback
- Minimizes wait time for unblocking playback
- Better UX - immediate feedback

### 4. Initial Segment Count
**Selected: 5 Completed Segments**

Pause for initial voice mapping after first 5 segments are completed.

**Rationale:**
- Clear and consistent metric
- Fast enough (30-60 seconds)
- Usually captures main personas in first chapter
- Not dependent on paragraph length

### 5. Voice Preview Audio
**Decision: Not Implemented Initially**

Voice preview generation (sample TTS for voice selection) is deferred to future work.

**Rationale:**
- Adds complexity and API costs
- Can rely on provider's voice descriptions initially
- Track as future enhancement

### 6. Multiple Unmapped Personas Handling
**Selected: Sequential Dialog Display**

Show one persona mapping dialog at a time, in order of first appearance.

**Rationale:**
- Simpler UX - user focuses on one character at a time
- Clear workflow - map one, move to next
- Natural order matches reading experience

### 7. Segment Order Preservation
**Selected: Prioritize Blocking Segments, Then Maintain Order**

When re-queueing newly mapped segments:
1. Place blocking segments (needed for current playback) at front
2. Place other segments in original order after current queue

**Rationale:**
- Minimizes playback interruption
- Maintains story flow
- Balances urgency with order

### 8. Skip/Cancel Option
**Selected: No Skip - Force Mapping**

User cannot skip voice mapping for a discovered persona.

**Rationale:**
- Quality control - ensures all audio is generated
- Prevents incomplete books
- Clear expectation - all personas must be mapped

---

## Data Flow Diagram

```
┌─────────────┐
│   Upload    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│    Parse    │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│  Segment 5 segments                                  │
│  Discover: [narrator, alice, witch]                 │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│  PAUSE: Show voice mapping dialog                    │
│  User maps: narrator→voice1, alice→voice2,          │
│             witch→voice3                             │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────────────┐
│  Start Parallel Pipeline:                            │
│  ┌───────────────────┐  ┌────────────────────────┐  │
│  │ Segmentation      │  │ TTS (mapped segments)  │  │
│  │ (background)      │→ │                        │  │
│  │                   │  │ Queue: seg1,2,3,4,5    │  │
│  └─────────┬─────────┘  └───────────┬────────────┘  │
│            │                        │                │
│            ▼                        ▼                │
│  ┌─────────────────┐     ┌──────────────────────┐  │
│  │ New segments:   │     │ Playback starts      │  │
│  │ seg6 [alice]    │     │ when audio ready     │  │
│  │ seg7 [bob]  ⚠️  │     │                      │  │
│  │ seg8 [narrator] │     │ Playing: seg1        │  │
│  └─────────┬───────┘     └──────────────────────┘  │
│            │                                         │
│            ▼                                         │
│  ┌──────────────────────────────────────┐          │
│  │ NEW PERSONA: "bob" discovered!       │          │
│  │ - seg7 → unmapped queue              │          │
│  │ - seg6, seg8 → mapped queue → TTS   │          │
│  │ - Show floating dialog for "bob"     │          │
│  └─────────┬────────────────────────────┘          │
│            │                                         │
│            ▼                                         │
│  User maps: bob → voice4                            │
│            │                                         │
│            ▼                                         │
│  ┌────────────────────────────────────────┐        │
│  │ Promote seg7 to front of TTS queue     │        │
│  │ Synthesize immediately                  │        │
│  │ Resume playback when ready              │        │
│  └─────────────────────────────────────────┘        │
└──────────────────────────────────────────────────────┘
```

---

## Edge Cases and Error Handling

### 1. Network Interruption During Processing
**Scenario:** User loses connection while pipeline is running

**Handling:**
- Backend pipeline continues independently
- Frontend shows "Reconnecting..." indicator
- On reconnect, fetch latest status and resume display
- No data loss - all state persisted to storage

### 2. TTS API Failure
**Scenario:** TTS provider returns error for specific segment

**Handling:**
- Retry up to 3 times with exponential backoff
- If still fails, mark segment as failed
- Continue processing other segments
- Show error indicator in UI with retry button
- Log failure for monitoring

### 3. Persona Discovery After All Segments Mapped
**Scenario:** Late-appearing persona discovered after user thinks mapping is complete

**Handling:**
- Same as normal: show floating dialog
- Pause synthesis of affected segments
- Wait for user mapping
- Resume with priority queue

### 4. User Closes Browser During Processing
**Scenario:** User navigates away or closes tab

**Handling:**
- Backend pipeline continues running
- When user returns, load current status
- Resume playback from where they left off
- Show updated progress for all stages

### 5. Segment with Multiple Personas
**Scenario:** LLM returns segment with dialogue from multiple characters

**Handling:**
- Current design: Each segment has one persona
- If this occurs, split segment in LLM prompt/response
- Fallback: Use first persona or "narrator"
- Log for monitoring and LLM prompt improvement

---

## Performance Considerations

### 1. Segmentation Speed
- **Bottleneck:** LLM API latency
- **Mitigation:** Batch processing (5 paragraphs per request)
- **Target:** 5 segments in 30-60 seconds

### 2. TTS Synthesis Speed
- **Bottleneck:** TTS API latency + concurrent limit
- **Mitigation:** 3 concurrent workers, queue management
- **Target:** 1 segment per 5-10 seconds (depends on length)

### 3. Frontend Polling
- **Overhead:** API calls every 2 seconds
- **Mitigation:** Only poll during active processing, stop when idle
- **Impact:** Minimal - status endpoint is lightweight

### 4. Audio Preloading
- **Challenge:** Balance between smooth playback and bandwidth
- **Strategy:** Preload next 3 segments (typically 30-90 seconds of audio)
- **Fallback:** Stream on-demand if preload fails

---

## Monitoring and Metrics

### Key Metrics to Track

1. **Pipeline Performance**
   - Time to first 5 segments
   - Time to initial voice mapping
   - Time to first playable audio
   - Total processing time per book

2. **User Behavior**
   - Time spent on voice mapping
   - Number of voice changes per persona
   - Playback interruptions (waiting for synthesis)
   - Drop-off points in flow

3. **API Usage**
   - LLM API calls and tokens
   - TTS API calls and characters
   - Retry rates for failed segments
   - Average cost per book

4. **Error Rates**
   - Segmentation failures
   - TTS synthesis failures
   - Network timeout rates
   - Recovery success rates

---

## Future Enhancements

### Short-term (Next Sprint)
1. Voice preview audio generation
2. Voice recommendation based on character description
3. WebSocket support for real-time updates
4. Audio caching and offline playback

### Medium-term (Next Quarter)
1. Multiple voice providers support
2. Voice cloning integration
3. Advanced voice effects (speed, pitch)
4. Batch book processing

### Long-term (Future)
1. AI-powered voice matching
2. Emotion-based voice modulation
3. Background music and sound effects
4. Multi-language support

---

## References

- Original Pipeline Orchestrator: `internal/pipeline/orchestrator.go`
- Voice Mapping API: `internal/api/book_handler.go`
- Segmentation Service: `internal/segmentation/service.go`
- TTS Orchestrator: `internal/tts/orchestrator.go`
- Book Types: `pkg/types/book.go`

---

## Change Log

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-30 | 1.0 | Initial design document | AI Assistant |

---

## Appendix: Design Decisions Q&A

### Q1: Why 5 segments instead of 10 or 20?
**A:** Balances speed (30-60s) with persona discovery. Most books introduce main characters in first few segments. Can be made configurable if needed.

### Q2: Why not batch multiple unmapped personas in one dialog?
**A:** Simpler UX - user focuses on one character at a time. Matches natural reading flow. Reduces cognitive load.

### Q3: Why not skip voice mapping entirely and use defaults?
**A:** Quality control. Ensures all audio meets user expectations. Prevents surprise voice changes. Maintains consistency.

### Q4: Why priority queue for newly mapped personas?
**A:** User is actively waiting for playback to resume. Minimizing wait time improves UX. Shows responsiveness to user action.

### Q5: Why polling instead of WebSocket?
**A:** Simplicity for MVP. Existing codebase uses polling. 2s latency acceptable. Can upgrade later without frontend changes.
