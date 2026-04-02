package domain

const RichEnvelopeFormatTipTap = "metaldocs.rich.tiptap"
const RichEnvelopeVersionV1 = 1

type RichEnvelope struct {
	Format  string         `json:"format"`
	Version int            `json:"version"`
	Content map[string]any `json:"content"`
}
