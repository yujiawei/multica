package agent

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestBuildQwenArgsBaseline(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("write a haiku", ExecOptions{}, slog.Default())
	expected := []string{
		"-p", "write a haiku",
		"-o", "stream-json",
		"--yolo",
		"-m", defaultQwenModel,
		"--auth-type", "openai",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, want := range expected {
		if args[i] != want {
			t.Fatalf("expected args[%d] = %q, got %q (args=%v)", i, want, args[i], args)
		}
	}
}

func TestBuildQwenArgsWithModelOverride(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{Model: "qwen3-coder-plus"}, slog.Default())

	var foundModel bool
	for i, a := range args {
		if a == "-m" {
			if i+1 >= len(args) || args[i+1] != "qwen3-coder-plus" {
				t.Fatalf("expected -m followed by qwen3-coder-plus, got %v", args)
			}
			foundModel = true
			break
		}
	}
	if !foundModel {
		t.Fatalf("expected -m flag, got args=%v", args)
	}
	// The default model must NOT also appear when an override is set.
	for i, a := range args {
		if a == "-m" && i+1 < len(args) && args[i+1] == defaultQwenModel {
			t.Fatalf("override should replace default model, got %v", args)
		}
	}
}

func TestBuildQwenArgsDefaultsModel(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{}, slog.Default())
	var foundDefault bool
	for i, a := range args {
		if a == "-m" && i+1 < len(args) && args[i+1] == defaultQwenModel {
			foundDefault = true
		}
	}
	if !foundDefault {
		t.Fatalf("expected -m %s when Model is empty, got %v", defaultQwenModel, args)
	}
}

func TestBuildQwenArgsWithSystemPrompt(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{SystemPrompt: "be terse"}, slog.Default())

	var found bool
	for i, a := range args {
		if a == "--append-system-prompt" {
			if i+1 >= len(args) || args[i+1] != "be terse" {
				t.Fatalf("expected --append-system-prompt followed by value, got %v", args)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --append-system-prompt flag when SystemPrompt is set, got %v", args)
	}
}

func TestBuildQwenArgsWithResume(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{ResumeSessionID: "abc-123"}, slog.Default())

	var found bool
	for i, a := range args {
		if a == "-r" {
			if i+1 >= len(args) || args[i+1] != "abc-123" {
				t.Fatalf("expected -r followed by session id, got %v", args)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected -r flag when ResumeSessionID is set, got %v", args)
	}
}

func TestBuildQwenArgsOmitsSystemPromptAndResumeWhenEmpty(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{}, slog.Default())
	for _, a := range args {
		if a == "--append-system-prompt" {
			t.Fatalf("expected no --append-system-prompt when empty, got %v", args)
		}
		if a == "-r" {
			t.Fatalf("expected no -r when ResumeSessionID empty, got %v", args)
		}
	}
}

func TestBuildQwenArgsDefaultsAuthTypeOpenAI(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{}, slog.Default())
	var found bool
	for i, a := range args {
		if a == "--auth-type" {
			if i+1 >= len(args) || args[i+1] != "openai" {
				t.Fatalf("expected --auth-type openai, got %v", args)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("expected default --auth-type openai, got %v", args)
	}
}

func TestBuildQwenArgsRespectsCustomAuthType(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{
		CustomArgs: []string{"--auth-type", "qwen-oauth"},
	}, slog.Default())

	// The daemon default must NOT be appended when the user supplies their own
	// --auth-type, and the user's value must survive.
	var openaiCount, oauthCount int
	for i, a := range args {
		if a == "--auth-type" && i+1 < len(args) {
			switch args[i+1] {
			case "openai":
				openaiCount++
			case "qwen-oauth":
				oauthCount++
			}
		}
	}
	if openaiCount != 0 {
		t.Fatalf("daemon default --auth-type openai should be suppressed, got %v", args)
	}
	if oauthCount != 1 {
		t.Fatalf("expected user's --auth-type qwen-oauth to survive once, got %v", args)
	}
}

func TestBuildQwenArgsRespectsCustomAuthTypeInlineForm(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{
		CustomArgs: []string{"--auth-type=qwen-oauth"},
	}, slog.Default())

	for i, a := range args {
		if a == "--auth-type" && i+1 < len(args) && args[i+1] == "openai" {
			t.Fatalf("inline --auth-type=... should suppress the daemon default, got %v", args)
		}
	}
}

func TestBuildQwenArgsPassesThroughCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{
		CustomArgs: []string{"--max-session-turns", "5"},
	}, slog.Default())

	if args[len(args)-2] != "--max-session-turns" || args[len(args)-1] != "5" {
		t.Fatalf("expected custom args at end, got %v", args)
	}
}

func TestBuildQwenArgsFiltersBlockedCustomArgs(t *testing.T) {
	t.Parallel()

	args := buildQwenArgs("hi", ExecOptions{
		CustomArgs: []string{"-o", "text", "--yolo", "--max-session-turns", "5"},
	}, slog.Default())

	// -o text and a duplicate --yolo should be filtered; --max-session-turns 5
	// passes through.
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && args[i+1] == "text" {
			t.Fatalf("blocked -o text should have been filtered: %v", args)
		}
	}
	if args[len(args)-2] != "--max-session-turns" || args[len(args)-1] != "5" {
		t.Fatalf("expected --max-session-turns 5 to pass through, got %v", args)
	}
}

func TestBuildQwenEnvSuppressesYoloWarningByDefault(t *testing.T) {
	t.Parallel()

	env := buildQwenEnv(nil)
	got, ok := envLookup(env, "QWEN_CODE_SUPPRESS_YOLO_WARNING")
	if !ok || got != "1" {
		t.Fatalf("expected QWEN_CODE_SUPPRESS_YOLO_WARNING=1, got %q (ok=%v)", got, ok)
	}
}

func TestBuildQwenEnvRespectsExplicitOverride(t *testing.T) {
	t.Parallel()

	env := buildQwenEnv(map[string]string{"QWEN_CODE_SUPPRESS_YOLO_WARNING": "0"})
	got, ok := envLookup(env, "QWEN_CODE_SUPPRESS_YOLO_WARNING")
	if !ok || got != "0" {
		t.Fatalf("expected caller override to win, got %q (ok=%v)", got, ok)
	}
}

func TestBuildQwenEnvPreservesOtherExtras(t *testing.T) {
	t.Parallel()

	env := buildQwenEnv(map[string]string{"OPENAI_BASE_URL": "https://gw.example/v1"})
	if got, ok := envLookup(env, "OPENAI_BASE_URL"); !ok || got != "https://gw.example/v1" {
		t.Fatalf("expected OPENAI_BASE_URL to pass through, got %q (ok=%v)", got, ok)
	}
	if got, ok := envLookup(env, "QWEN_CODE_SUPPRESS_YOLO_WARNING"); !ok || got != "1" {
		t.Fatalf("expected default suppress flag, got %q (ok=%v)", got, ok)
	}
}

// TestQwenStreamParsingReusesClaude verifies that qwen's real stream-json
// output — which is the Claude schema — is parsed correctly by the reused
// claudeBackend handlers and claudeResultUsage. The lines below are verbatim
// envelopes captured from `qwen -p ... -o stream-json` 0.17.1.
func TestQwenStreamParsingReusesClaude(t *testing.T) {
	t.Parallel()

	lines := []string{
		`{"type":"system","subtype":"init","session_id":"sess-1","model":"qwen3.7-max"}`,
		`{"type":"assistant","session_id":"sess-1","message":{"role":"assistant","model":"qwen3.7-max","content":[{"type":"thinking","thinking":"hm"}],"usage":{"input_tokens":0,"output_tokens":0}}}`,
		`{"type":"assistant","session_id":"sess-1","message":{"role":"assistant","model":"qwen3.7-max","content":[{"type":"text","text":"Hello!"}],"usage":{"input_tokens":18607,"output_tokens":11,"cache_read_input_tokens":0}}}`,
		`{"type":"result","subtype":"success","session_id":"sess-1","is_error":false,"result":"Hello!","usage":{"input_tokens":28090,"output_tokens":139,"cache_read_input_tokens":0}}`,
	}

	cb := &claudeBackend{cfg: Config{Logger: slog.Default()}}
	ch := make(chan Message, 64)
	var output strings.Builder
	usage := make(map[string]TokenUsage)
	var sessionID string
	var finalStatus = "completed"

	for _, line := range lines {
		var msg claudeSDKMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		switch msg.Type {
		case "system":
			sessionID = msg.SessionID
		case "assistant":
			cb.handleAssistant(msg, ch, &output, usage)
		case "user":
			cb.handleUser(msg, ch)
		case "result":
			if msg.SessionID != "" {
				sessionID = msg.SessionID
			}
			if msg.ResultText != "" {
				output.Reset()
				output.WriteString(msg.ResultText)
			}
			if ru := claudeResultUsage(msg, qwenModel(ExecOptions{})); len(ru) > 0 {
				usage = ru
			}
			if msg.IsError {
				finalStatus = "failed"
			}
		}
	}

	if sessionID != "sess-1" {
		t.Fatalf("expected session id sess-1, got %q", sessionID)
	}
	if finalStatus != "completed" {
		t.Fatalf("expected completed, got %q", finalStatus)
	}
	if output.String() != "Hello!" {
		t.Fatalf("expected output 'Hello!', got %q", output.String())
	}
	u, ok := usage["qwen3.7-max"]
	if !ok {
		t.Fatalf("expected usage keyed by qwen3.7-max, got %v", usage)
	}
	if u.InputTokens != 28090 || u.OutputTokens != 139 {
		t.Fatalf("expected result usage 28090/139, got %+v", u)
	}

	// Verify the streamed messages: thinking, then text.
	var sawThinking, sawText bool
	for {
		select {
		case m := <-ch:
			switch m.Type {
			case MessageThinking:
				sawThinking = true
			case MessageText:
				if m.Content == "Hello!" {
					sawText = true
				}
			}
			continue
		default:
		}
		break
	}
	if !sawThinking {
		t.Fatalf("expected a thinking message")
	}
	if !sawText {
		t.Fatalf("expected a text message with 'Hello!'")
	}
}

func TestQwenResultErrorText(t *testing.T) {
	t.Parallel()

	// Real failure envelope: result empty, message under error.message.
	line := `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"","error":{"message":"No auth type is selected."}}`
	got := qwenResultErrorText(line, "")
	if got != "No auth type is selected." {
		t.Fatalf("expected nested error message, got %q", got)
	}

	// Fallback to resultText when no nested error object.
	got = qwenResultErrorText(`{"type":"result","is_error":true,"result":"boom"}`, "boom")
	if got != "boom" {
		t.Fatalf("expected fallback to result text, got %q", got)
	}
}

func TestQwenModelDefault(t *testing.T) {
	t.Parallel()

	if got := qwenModel(ExecOptions{}); got != defaultQwenModel {
		t.Fatalf("expected default %q, got %q", defaultQwenModel, got)
	}
	if got := qwenModel(ExecOptions{Model: "qwen3-max"}); got != "qwen3-max" {
		t.Fatalf("expected override, got %q", got)
	}
}

func TestQwenBackendRegistered(t *testing.T) {
	t.Parallel()

	b, err := New("qwen", Config{Logger: slog.Default()})
	if err != nil {
		t.Fatalf("New(qwen) failed: %v", err)
	}
	if _, ok := b.(*qwenBackend); !ok {
		t.Fatalf("expected *qwenBackend, got %T", b)
	}
}
