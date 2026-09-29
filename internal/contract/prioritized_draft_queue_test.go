package contract_test

import (
	"testing"

	"portaljuridico/internal/editorialdrafts"
	"portaljuridico/internal/reviewqueue"
)

func TestPrioritizedDigitalSeedsPersistDraftsAndQueueWithoutPublishing(t *testing.T) {
	drafts, draftReport := editorialdrafts.LoadRecords(".")
	if !draftReport.Passed() {
		t.Fatalf("could not load editorial drafts: %v", draftReport.Messages())
	}
	queue, queueReport := reviewqueue.LoadRecords(".")
	if !queueReport.Passed() {
		t.Fatalf("could not load review queue: %v", queueReport.Messages())
	}

	for _, termID := range []string{
		"advogado-trabalhista-online",
		"negativa-cobertura-plano-saude",
		"divorcio-online",
		"auxilio-doenca-negado",
		"inventario-extrajudicial-online",
		"desconto-indevido-inss",
	} {
		draft := findDraft(drafts, termID)
		if draft == nil {
			t.Fatalf("missing persisted draft for prioritized term %q", termID)
		}
		if draft.Status != "draft" || draft.IndexPolicy != "noindex" || draft.PublicPath != "" {
			t.Fatalf("draft %q escaped lab contract: status=%s index=%s public_path=%q", termID, draft.Status, draft.IndexPolicy, draft.PublicPath)
		}
		record := findQueueRecord(queue, termID)
		if record == nil {
			t.Fatalf("missing review queue record for prioritized term %q", termID)
		}
		if record.PublicationAllowed || record.PublicPath != "" || record.ReviewStatus != "needs_review" {
			t.Fatalf("queue record %q escaped review contract: status=%s publication=%t public_path=%q", termID, record.ReviewStatus, record.PublicationAllowed, record.PublicPath)
		}
	}
}

func findDraft(records []editorialdrafts.Record, termID string) *editorialdrafts.Record {
	for i := range records {
		if records[i].TermID == termID {
			return &records[i]
		}
	}
	return nil
}

func findQueueRecord(records []reviewqueue.Record, termID string) *reviewqueue.Record {
	for i := range records {
		if records[i].TermID == termID {
			return &records[i]
		}
	}
	return nil
}
