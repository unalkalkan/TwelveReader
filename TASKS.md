# Tasks

## TODO

- [ ] Present available characters in each segmentation request (batch and single), so that new segmentation runs would not generate similar but not the same characters. Here's the following people list from the Suskind's Perfume. We need to eliminate duplicate people. Instruct the LLM in system prompt about this situation, that it can create people if they are not present in the list that is provided.

```
Father Terrier
Father_Terrier
Father_Terrier (fantasy/inner)
Father_Terrier (inner/exclaimed)
Father_Terrier_thought
Grenouille
Horace (quoted)
Jeanne_Bussie
Terrier
Terrier (spoken)
Terrier (thought)
bystander
character1
character_grenouille
dialogue_speaker
monk
narrator
other_woman
title_front_matter
wet nurse
wet_nurse
wet_nurse_Jeanne_Bussie
woman
```

- [ ] When the first X segments received, proceed with the TTS, and then proceed to play the book. Orchestation on LLM->TTS->Read should handle in parallel so that the user would not wait for the whole book to be finished before running the audio.

- [ ] Context deadlines may exceed way too quickly right now, we should raise these deadline timeouts. Below log is from the TTS, but I've seen that it also happens in the LLM.
```
2026/01/29 10:49:56 [TTS-qwen3-tts] Request failed after 2m0.002253724s: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 10:49:56 Failed to synthesize segment seg_00035: TTS provider failed: failed to call TTS API: failed to execute request: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

- [ ] Advanced retrying and error handling on Segmentation and Synthesization. Make sure that the errored segments/syths are requeued.
- [ ] I believe the Synthesization is NOT done parallel right now. I believe lock mechanisms disable the paralleling capabilities. Investigate this.
- [ ] Log segment name in
- [ ] Ability to save the segmentation progress and continue where it's left of.
- [ ] ePUB/PDF implementation
- [ ] OCR implementation
- [ ] Improved TXT implementation
- [ ] We should figure out how to establish a consistent tone when using the narrator voice. Right now, due to the changes in the voice descriptions, each paragraph/sentence of the narrator might change a tone shift so much that the voice can feel that it belongs to a different person.
- [ ] Should we worry about consistent tones between sentences of people? On top of my head -- maybe we can ensure a stable tone if the segments/senteces follow each other. But that might not always work. One other idea is to request the LLM to keep a stable tone/voice description with giving previous segments.
- [ ] UUF - A GREAT IDEA - We can add characteristis to the people of the book. They can persist across different voice run types, and the hints would change the tone of the voice. Build up a persistent character of the voice, attach it to the every run.


- [ ] Overall failed:
```
2026/01/29 11:20:30 Failed to synthesize segment seg_00144: TTS provider failed: failed to call TTS API: failed to execute request: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 11:20:30 [TTS-qwen3-tts] Request: POST http://192.168.2.113:8000/v1/audio/speech
2026/01/29 11:20:30 [TTS-qwen3-tts] Request payload: model=qwen3-tts-customvoice-1.7b, voice=ryan, input_length=421 chars
2026/01/29 11:20:30 [TTS-qwen3-tts] Request input (truncated): ‘Because he’s healthy,’ Terrier cried, ‘because he’s healthy, that’s why he doesn’t smell! Only sick babies smell, everyone knows that. It’s well known that a child with the pox smells... (truncated)
2026/01/29 11:20:48 [TTS-qwen3-tts] Request failed after 2m0.000225773s: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 11:20:48 Failed to synthesize segment seg_00138: TTS provider failed: failed to call TTS API: failed to execute request: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 11:21:55 [TTS-qwen3-tts] Request failed after 2m0.000760111s: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 11:21:55 Failed to synthesize segment seg_00143: TTS provider failed: failed to call TTS API: failed to execute request: Post "http://192.168.2.113:8000/v1/audio/speech": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
2026/01/29 11:22:13 [TTS-qwen3-tts] Response: 200 200 OK (took 1m43.385477802s)
2026/01/29 11:22:13 [TTS-qwen3-tts] Response payload: audio_size=1499264 bytes
2026/01/29 11:22:13 TTS synthesis failed for book book_1769682146657816902: synthesis completed with 25 errors out of 145 segments
```

## DONE