import { z } from 'zod'

// Book Status
export const BookStatusSchema = z.enum([
  'uploaded',
  'parsing',
  'segmenting',
  'voice_mapping',
  'ready',
  'synthesizing',
  'synthesized',
  'synthesis_error',
  'error',
])

export type BookStatus = z.infer<typeof BookStatusSchema>

// Book Metadata
export const BookMetadataSchema = z.object({
  id: z.string(),
  title: z.string(),
  author: z.string(),
  language: z.string(),
  uploaded_at: z.string(),
  status: BookStatusSchema,
  orig_format: z.string(),
  total_chapters: z.number(),
  total_segments: z.number(),
})

export type BookMetadata = z.infer<typeof BookMetadataSchema>

// Processing Status
export const ProcessingStatusSchema = z.object({
  book_id: z.string(),
  status: BookStatusSchema,
  stage: z.string(),
  progress: z.number(),
  total_chapters: z.number(),
  parsed_chapters: z.number(),
  total_segments: z.number(),
  updated_at: z.string(),
})

export type ProcessingStatus = z.infer<typeof ProcessingStatusSchema>

// Segment
export const SegmentSchema = z.object({
  id: z.string(),
  book_id: z.string(),
  chapter: z.string(),
  toc_path: z.array(z.string()),
  text: z.string(),
  language: z.string(),
  person: z.string(),
  voice_description: z.string(),
  processing: z.object({
    segmenter_version: z.string(),
    generated_at: z.string(),
  }),
  timestamps: z
    .object({
      precision: z.enum(['word', 'sentence']),
      items: z.array(
        z.object({
          word: z.string(),
          start: z.number(),
          end: z.number(),
        })
      ),
    })
    .optional(),
  audio_url: z.string().optional(),
})

export type Segment = z.infer<typeof SegmentSchema>

// Voice Map
export const PersonVoiceSchema = z.object({
  id: z.string(),
  provider_voice: z.string(),
})

export const VoiceMapSchema = z.object({
  book_id: z.string(),
  persons: z.array(PersonVoiceSchema),
})

export type PersonVoice = z.infer<typeof PersonVoiceSchema>
export type VoiceMap = z.infer<typeof VoiceMapSchema>

// Server Info
export const ServerInfoSchema = z.object({
  version: z.string(),
  storage_adapter: z.string(),
})

export type ServerInfo = z.infer<typeof ServerInfoSchema>

// Providers
export const ProvidersSchema = z.object({
  llm: z.array(z.string()),
  tts: z.array(z.string()),
  ocr: z.array(z.string()),
})

export type Providers = z.infer<typeof ProvidersSchema>
