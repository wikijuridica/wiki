package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"portaljuridico/internal/sources"
	"portaljuridico/internal/storage"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	registry, err := sources.LoadRegistry(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 12 * time.Second}
	// Audit-only fallback: a TLS-unverified status is recorded as blocked evidence, never as approval.
	insecureAuditClient := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	checkedAt := time.Now().Format("2006-01-02")
	checkedAtTime := time.Now().Format(time.RFC3339)
	failures := make([]string, 0)
	for i := range registry.Sources {
		source := &registry.Sources[i]
		robotsStatus, err := auditURL(client, insecureAuditClient, source.RobotsURL)
		if err != nil {
			robotsStatus = "network_error"
			failures = append(failures, source.SourceID+":robots:"+err.Error())
		}
		termsStatus, err := auditURL(client, insecureAuditClient, source.TermsURL)
		if err != nil {
			termsStatus = "network_error"
			failures = append(failures, source.SourceID+":terms:"+err.Error())
		}

		source.RobotsStatus = robotsStatus
		source.RobotsCheckedAt = checkedAt
		source.TermsStatus = termsStatus
		source.TermsCheckedAt = checkedAt
		source.AuditDecision = auditDecision(robotsStatus, termsStatus)
		source.AuditNote = "Auditoria HTTP live de robots e termos executada sem ingestao de conteudo; status prova alcance, nao aprova coleta, publicacao ou scraping."
		source.IngestionEnabled = false
		if err := storage.AppendJSONL(root, "source_audits", sourceAuditEvent{
			CheckedAt:        checkedAtTime,
			SourceID:         source.SourceID,
			RobotsURL:        source.RobotsURL,
			RobotsStatus:     robotsStatus,
			TermsURL:         source.TermsURL,
			TermsStatus:      termsStatus,
			AuditDecision:    source.AuditDecision,
			IngestionEnabled: false,
			Note:             source.AuditNote,
		}); err != nil {
			failures = append(failures, source.SourceID+":storage:"+err.Error())
		}
	}

	if err := writeRegistry(registry); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, source := range registry.Sources {
		fmt.Printf("%s robots=%s terms=%s decision=%s\n", source.SourceID, source.RobotsStatus, source.TermsStatus, source.AuditDecision)
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Println("failure=" + failure)
		}
		os.Exit(1)
	}
}

type sourceAuditEvent struct {
	CheckedAt        string `json:"checked_at"`
	SourceID         string `json:"source_id"`
	RobotsURL        string `json:"robots_url"`
	RobotsStatus     string `json:"robots_status"`
	TermsURL         string `json:"terms_url"`
	TermsStatus      string `json:"terms_status"`
	AuditDecision    string `json:"audit_decision"`
	IngestionEnabled bool   `json:"ingestion_enabled"`
	Note             string `json:"note"`
}

func auditURL(client *http.Client, insecureAuditClient *http.Client, url string) (string, error) {
	status, err := auditURLWithClient(client, url)
	if err == nil {
		return status, nil
	}
	if isUnknownAuthority(err) {
		fallbackStatus, fallbackErr := auditURLWithClient(insecureAuditClient, url)
		if fallbackErr != nil {
			return "", fmt.Errorf("tls_unknown_authority_then_fallback_failed:%w", fallbackErr)
		}
		return "tls_unverified_" + fallbackStatus, nil
	}
	return "", err
}

func auditURLWithClient(client *http.Client, url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("empty_url")
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "PortalJuridicoAudit/0.1 source-contract-check")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	_, _ = io.CopyN(io.Discard, response.Body, 1024)
	return fmt.Sprintf("http_%d", response.StatusCode), nil
}

func auditDecision(robotsStatus string, termsStatus string) string {
	if statusOK(robotsStatus) && statusOK(termsStatus) {
		return "audited_reachable_still_blocked"
	}
	if statusTLSFallbackOK(robotsStatus) || statusTLSFallbackOK(termsStatus) {
		return "audited_reachable_with_security_or_policy_restriction_still_blocked"
	}
	return "audited_unreachable_or_restricted_still_blocked"
}

func statusOK(status string) bool {
	return strings.HasPrefix(status, "http_2") || strings.HasPrefix(status, "http_3")
}

func statusTLSFallbackOK(status string) bool {
	return strings.HasPrefix(status, "tls_unverified_http_2") || strings.HasPrefix(status, "tls_unverified_http_3")
}

func isUnknownAuthority(err error) bool {
	var unknownAuthority x509.UnknownAuthorityError
	return errors.As(err, &unknownAuthority)
}

func writeRegistry(registry sources.Registry) error {
	data, err := json.MarshalIndent(struct {
		Sources []sources.Source `json:"sources"`
	}{Sources: registry.Sources}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(registry.RegistryPath(), data, 0644)
}
