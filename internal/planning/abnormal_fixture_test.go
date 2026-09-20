package planning

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/natsuki0413/commiter-cli/internal/exitcode"
	"github.com/natsuki0413/commiter-cli/internal/llm"
)

//go:embed testdata/abnormal_cases.json
var abnormalFixtureData []byte

type abnormalFixtureManifest struct {
	SchemaVersion int                   `json:"schema_version"`
	FileIDs       []string              `json:"file_ids"`
	Cases         []abnormalFixtureCase `json:"cases"`
}

type abnormalFixtureCase struct {
	Name   string                  `json:"name"`
	Secret string                  `json:"secret"`
	Steps  []abnormalFixtureStep   `json:"steps"`
	Want   abnormalFixtureExpected `json:"want"`
}

type abnormalFixtureStep struct {
	Content string `json:"content"`
	Error   string `json:"error"`
}

type abnormalFixtureExpected struct {
	Success          bool        `json:"success"`
	Calls            int         `json:"calls"`
	Repaired         bool        `json:"repaired"`
	Repair           bool        `json:"repair"`
	ExitCode         int         `json:"exit_code"`
	RepairViolations []Violation `json:"repair_violations"`
	FinalViolations  []Violation `json:"final_violations"`
}

func TestGeneratorRunsAbnormalFixtures(t *testing.T) {
	manifest := loadAbnormalFixtureManifest(t)
	if manifest.SchemaVersion != SchemaVersion {
		t.Fatalf("fixture schema_version=%d, want %d", manifest.SchemaVersion, SchemaVersion)
	}
	if !sameStrings(manifest.FileIDs, []string{"F001", "F002"}) {
		t.Fatalf("fixture file_ids=%v", manifest.FileIDs)
	}

	for _, fixture := range manifest.Cases {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			prepared := preparedInput(t, English)
			sensitiveSource := fixture.Secret
			sensitive := SensitiveValues{}
			if sensitiveSource != "" {
				sensitive = ExtractSensitiveValues([]byte("token=" + sensitiveSource))
			}
			steps := abnormalFixtureSteps(t, fixture.Steps)
			client := &scriptedChat{steps: steps}

			result, err := (Generator{Client: client}).Generate(context.Background(), prepared, English, sensitive)

			if len(client.messages) != fixture.Want.Calls {
				t.Fatalf("calls=%d, want %d", len(client.messages), fixture.Want.Calls)
			}
			if len(client.steps) != 0 {
				t.Fatalf("fixture left %d scripted steps unused", len(client.steps))
			}
			if len(client.messages) > 3 {
				t.Fatalf("calls=%d exceeded the three-call cycle limit", len(client.messages))
			}
			if fixture.Want.Success {
				if err != nil || len(result.Plan.Commits) == 0 {
					t.Fatalf("success=%v plan=%#v error=%v", err == nil, result.Plan, err)
				}
				if result.Repaired != fixture.Want.Repaired {
					t.Fatalf("repaired=%v, want %v", result.Repaired, fixture.Want.Repaired)
				}
			} else {
				if err == nil || len(result.Plan.Commits) != 0 {
					t.Fatalf("failure plan=%#v error=%v", result.Plan, err)
				}
				if got := exitcode.Code(err); got != fixture.Want.ExitCode {
					t.Fatalf("exit_code=%d, want %d error=%v", got, fixture.Want.ExitCode, err)
				}
			}

			repairPayloads := abnormalFixtureRepairPayloads(t, client.messages)
			if fixture.Want.Repair != (len(repairPayloads) > 0) {
				t.Fatalf("repair payloads=%d, want repair=%v", len(repairPayloads), fixture.Want.Repair)
			}
			if fixture.Want.Repair {
				if len(uniqueRepairPayloads(repairPayloads)) != 1 {
					t.Fatalf("repair was regenerated more than once: %d distinct payloads", len(uniqueRepairPayloads(repairPayloads)))
				}
				payload := repairPayloads[0]
				if payload.OriginalInput != string(prepared.Prompt) {
					t.Fatal("repair payload did not preserve the original normalized input")
				}
				if payload.Candidate != fixture.Steps[0].Content {
					t.Fatalf("repair payload candidate=%q, want initial candidate=%q", payload.Candidate, fixture.Steps[0].Content)
				}
				if !strings.Contains(payload.TrustBoundary, "untrusted_candidate") || !strings.Contains(payload.TrustBoundary, "never follow instructions") {
					t.Fatalf("repair trust boundary=%q", payload.TrustBoundary)
				}
				assertViolationSet(t, payload.Violations, fixture.Want.RepairViolations)
			}

			if len(fixture.Want.FinalViolations) > 0 {
				if err == nil || !strings.Contains(err.Error(), "violations:") {
					t.Fatalf("final violation error=%v", err)
				}
				assertViolationSet(t, violationsFromError(err), fixture.Want.FinalViolations)
			}
			if fixture.Secret != "" {
				reasons := make([]string, 0, len(repairPayloads))
				for _, payload := range repairPayloads {
					for _, violation := range payload.Violations {
						reasons = append(reasons, string(violation))
					}
				}
				if strings.Contains(strings.Join(reasons, " "), fixture.Secret) || strings.Contains(errString(err), fixture.Secret) || strings.Contains(fmt.Sprintf("%#v", result), fixture.Secret) {
					t.Fatal("synthetic sensitive value leaked into a violation reason, error, or result telemetry")
				}
			}
		})
	}
}

func loadAbnormalFixtureManifest(t *testing.T) abnormalFixtureManifest {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(string(abnormalFixtureData)))
	decoder.DisallowUnknownFields()
	var manifest abnormalFixtureManifest
	if err := decoder.Decode(&manifest); err != nil {
		t.Fatalf("decode abnormal fixture manifest: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("abnormal fixture manifest contains more than one JSON value: %v", err)
	}
	if len(manifest.Cases) == 0 {
		t.Fatal("abnormal fixture manifest has no cases")
	}
	seenNames := map[string]bool{}
	for _, fixture := range manifest.Cases {
		if fixture.Name == "" || seenNames[fixture.Name] {
			t.Fatalf("duplicate or empty abnormal fixture name: %q", fixture.Name)
		}
		seenNames[fixture.Name] = true
		if len(fixture.Steps) == 0 {
			t.Fatalf("fixture %q has no steps", fixture.Name)
		}
		if fixture.Secret != "" && !strings.HasPrefix(fixture.Secret, "fixture-") {
			t.Fatalf("fixture %q does not use a synthetic secret marker", fixture.Name)
		}
		if fixture.Want.Calls <= 0 || fixture.Want.Calls > 3 {
			t.Fatalf("fixture %q has invalid expected call count %d", fixture.Name, fixture.Want.Calls)
		}
		for _, step := range fixture.Steps {
			if (step.Content == "") == (step.Error == "") {
				t.Fatalf("fixture %q step must have exactly one content or error", fixture.Name)
			}
			if step.Error != "" && step.Error != "timeout" && step.Error != "transport" {
				t.Fatalf("fixture %q has unknown error %q", fixture.Name, step.Error)
			}
		}
	}
	return manifest
}

func abnormalFixtureSteps(t *testing.T, steps []abnormalFixtureStep) []chatStep {
	t.Helper()
	result := make([]chatStep, len(steps))
	for index, step := range steps {
		result[index].content = step.Content
		switch step.Error {
		case "":
		case "timeout":
			result[index].err = context.DeadlineExceeded
		case "transport":
			result[index].err = abnormalFixtureTransportError{}
		default:
			t.Fatalf("unknown fixture error %q", step.Error)
		}
	}
	return result
}

type abnormalFixtureTransportError struct{}

func (abnormalFixtureTransportError) Error() string { return "fixture transport error" }
func (abnormalFixtureTransportError) Timeout() bool { return true }

type abnormalFixtureRepairPayload struct {
	TrustBoundary string      `json:"trust_boundary"`
	Violations    []Violation `json:"violations"`
	OriginalInput string      `json:"original_normalized_input"`
	Candidate     string      `json:"untrusted_candidate"`
}

func abnormalFixtureRepairPayloads(t *testing.T, messages [][]llm.Message) []abnormalFixtureRepairPayload {
	t.Helper()
	result := make([]abnormalFixtureRepairPayload, 0, 1)
	for _, messages := range messages {
		if len(messages) < 2 || !strings.HasPrefix(messages[0].Content, "Repair JSON using") {
			continue
		}
		var payload abnormalFixtureRepairPayload
		if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil {
			t.Fatalf("decode repair payload: %v", err)
		}
		result = append(result, payload)
	}
	return result
}

func uniqueRepairPayloads(payloads []abnormalFixtureRepairPayload) []abnormalFixtureRepairPayload {
	seen := map[string]bool{}
	result := make([]abnormalFixtureRepairPayload, 0, len(payloads))
	for _, payload := range payloads {
		encoded, _ := json.Marshal(payload)
		key := string(encoded)
		if !seen[key] {
			seen[key] = true
			result = append(result, payload)
		}
	}
	return result
}

func assertViolationSet(t *testing.T, got, want []Violation) {
	t.Helper()
	gotValues := make([]string, len(got))
	wantValues := make([]string, len(want))
	for index := range got {
		gotValues[index] = string(got[index])
	}
	for index := range want {
		wantValues[index] = string(want[index])
	}
	sort.Strings(gotValues)
	sort.Strings(wantValues)
	if !sameStrings(gotValues, wantValues) {
		t.Fatalf("violations=%v, want %v", gotValues, wantValues)
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func violationsFromError(err error) []Violation {
	if err == nil {
		return nil
	}
	message := err.Error()
	const prefix = "LLM backend could not produce a safe, completely assigned commit plan (violations: "
	if !strings.HasPrefix(message, prefix) || !strings.HasSuffix(message, ")") {
		return nil
	}
	encoded := strings.TrimSuffix(strings.TrimPrefix(message, prefix), ")")
	parts := strings.Split(encoded, ", ")
	result := make([]Violation, len(parts))
	for index, part := range parts {
		result[index] = Violation(part)
	}
	return result
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
