package assistant

import (
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
)

// SetAfterAnswer runs f once the provider answered a turn, before its
// checkpoint.
func SetAfterAnswer(s *Service, f func(turn uuid.UUID)) { s.afterAnswer = f }

// SetClaimHook runs f in a claim's transaction: once its locks are taken
// ("locked"), and once its slot is ("slot").
func SetClaimHook(s *Service, f func(turn uuid.UUID, stage string)) { s.inClaim = f }

// DeltaBatch is a turn's answering state and a piece of its answer.
type DeltaBatch struct {
	Workspace, Thread, Turn uuid.UUID
	Version                 int64
	Text                    string
}

// FlushDeltas writes batches as one group, as the group writer does, and
// returns each one's outcome: written, fenced, skipped (the group writer
// writes it later) or failed.
func FlushDeltas(s *Service, in []DeltaBatch) []string {
	ws := make([]*deltaWriter, len(in))
	for i, b := range in {
		w := s.deltaWriter(b.Workspace, b.Thread, b.Turn, b.Version, func(error) {})
		w.firstAt = time.Now()
		w.buf.WriteString(b.Text)
		ws[i] = w
	}
	names := map[batchStatus]string{batchWritten: "written", batchFenced: "fenced", batchSkipped: "skipped", batchFailed: "failed"}
	out := make([]string, len(in))
	for _, b := range ws[0].g.flush(ws) {
		for i, w := range ws {
			if b.w == w {
				out[i] = names[b.status]
			}
		}
	}
	return out
}

// DeltaWriter is a turn's writer: Emit hands it a piece as a provider
// streams one; Take ends it and returns what wasn't written.
type DeltaWriter struct{ w *deltaWriter }

func NewDeltaWriter(s *Service, b DeltaBatch) *DeltaWriter {
	return &DeltaWriter{w: s.deltaWriter(b.Workspace, b.Thread, b.Turn, b.Version, func(error) {})}
}

func (d *DeltaWriter) Emit(text string) error { return d.w.emit(llm.Delta{Text: text}) }

func (d *DeltaWriter) Take() string { return d.w.take() }

// SetDeltaGroupWait sets how long the group writer gathers batches, until
// restore.
func SetDeltaGroupWait(d time.Duration) (restore func()) {
	groupWaitSet.Store(int64(d))
	return func() { groupWaitSet.Store(0) }
}
