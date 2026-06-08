package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// qwenBackend implements Backend by spawning the Qwen Code CLI
// (https://github.com/QwenLM/qwen-code) in non-interactive mode with
// `-o stream-json`.
//
// Qwen Code is a gemini-cli fork, so its CLI *arguments* follow the gemini
// shape (`-p <prompt> -o stream-json --yolo -m <model>` etc.), but its
// streaming *output* is already the Claude Code stream-json schema
// (`type:"assistant"` envelopes carrying `message.content[]` blocks plus
// `usage.input_tokens`). We therefore build gemini-style argv here and reuse
// claudeBackend's handleAssistant / handleUser / claudeResultUsage parsing via
// composition rather than re-implementing the Claude schema.
//
// Unlike Claude (which speaks a bidirectional stream-json protocol over stdin),
// Qwen takes the whole prompt as a single `-p` argument and never reads control
// frames back, so this backend has no stdin writer and no control_request
// handling — it mirrors gemini.go's one-shot stdout-scanning lifecycle.
type qwenBackend struct {
	cfg Config
}

// defaultQwenModel is used when neither the agent config nor the per-run
// options pick a model. The mlamp gateway exposes qwen3.7-max on its OpenAI
// endpoint; the daemon-wide default can still be overridden via
// MULTICA_QWEN_MODEL (agent.model) or a per-task model selection.
const defaultQwenModel = "qwen3.7-max"

func (b *qwenBackend) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Session, error) {
	execPath := b.cfg.ExecutablePath
	if execPath == "" {
		execPath = "qwen"
	}
	if _, err := exec.LookPath(execPath); err != nil {
		return nil, fmt.Errorf("qwen executable not found at %q: %w", execPath, err)
	}

	timeout := opts.Timeout
	runCtx, cancel := runContext(ctx, timeout)

	args := buildQwenArgs(prompt, opts, b.cfg.Logger)

	cmd := exec.CommandContext(runCtx, execPath, args...)
	hideAgentWindow(cmd)
	b.cfg.Logger.Info("agent command", "exec", execPath, "args", args)
	cmd.WaitDelay = 10 * time.Second
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = buildQwenEnv(b.cfg.Env)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("qwen stdout pipe: %w", err)
	}
	// Capture stderr into both the daemon log and a bounded tail buffer so we
	// can attach the last few KB to Result.Error when qwen exits unexpectedly
	// (auth not configured, gateway 4xx/5xx, …). Mirrors claude.go.
	stderrBuf := newStderrTail(newLogWriter(b.cfg.Logger, "[qwen:stderr] "), agentStderrTailBytes)
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start qwen: %w", err)
	}

	b.cfg.Logger.Info("qwen started", "pid", cmd.Process.Pid, "cwd", opts.Cwd, "model", opts.Model)

	msgCh := make(chan Message, 256)
	resCh := make(chan Result, 1)

	// Close stdout when the context is cancelled so scanner.Scan() unblocks.
	go func() {
		<-runCtx.Done()
		_ = stdout.Close()
	}()

	// Reuse claudeBackend's stream parsing: qwen's output is the Claude
	// stream-json schema, so handleAssistant / handleUser / claudeResultUsage
	// apply verbatim. Composition (not copy-paste) keeps the two backends in
	// lockstep if the Claude schema parsing ever changes.
	cb := &claudeBackend{cfg: b.cfg}

	go func() {
		defer cancel()
		defer close(msgCh)
		defer close(resCh)

		startTime := time.Now()
		var output strings.Builder
		var sessionID string
		finalStatus := "completed"
		var finalError string
		usage := make(map[string]TokenUsage)

		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			var msg claudeSDKMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "system":
				// Qwen emits a `type:"system" subtype:"init"` envelope first,
				// carrying the session_id we report for resume.
				if msg.SessionID != "" {
					sessionID = msg.SessionID
				}
				trySend(msgCh, Message{Type: MessageStatus, Status: "running", SessionID: sessionID})
			case "assistant":
				cb.handleAssistant(msg, msgCh, &output, usage)
			case "user":
				cb.handleUser(msg, msgCh)
			case "result":
				if msg.SessionID != "" {
					sessionID = msg.SessionID
				}
				if msg.ResultText != "" {
					output.Reset()
					output.WriteString(msg.ResultText)
				}
				if resultUsage := claudeResultUsage(msg, qwenModel(opts)); len(resultUsage) > 0 {
					usage = resultUsage
				}
				if msg.IsError {
					finalStatus = "failed"
					// On error, qwen leaves the `result` field empty and carries
					// the message under a nested `error.message` instead; parse
					// it from the raw line, falling back to ResultText.
					finalError = qwenResultErrorText(line, msg.ResultText)
				}
			}
		}

		waitErr := cmd.Wait()
		duration := time.Since(startTime)

		switch {
		case runCtx.Err() == context.DeadlineExceeded:
			finalStatus = "timeout"
			finalError = fmt.Sprintf("qwen timed out after %s", timeout)
		case runCtx.Err() == context.Canceled:
			finalStatus = "aborted"
			finalError = "execution cancelled"
		case waitErr != nil && finalStatus == "completed":
			finalStatus = "failed"
			finalError = fmt.Sprintf("qwen exited with error: %v", waitErr)
		}

		// cmd.Wait() has returned, so os/exec has copied every stderr byte —
		// the tail is safe to sample. Attach it to any non-empty failure.
		if finalError != "" {
			finalError = withAgentStderr(finalError, "qwen", stderrBuf.Tail())
		}

		b.cfg.Logger.Info("qwen finished", "pid", cmd.Process.Pid, "status", finalStatus, "duration", duration.Round(time.Millisecond).String())

		reportedSessionID := resolveSessionID(opts.ResumeSessionID, sessionID, finalStatus == "failed")

		resCh <- Result{
			Status:     finalStatus,
			Output:     output.String(),
			Error:      finalError,
			DurationMs: duration.Milliseconds(),
			SessionID:  reportedSessionID,
			Usage:      usage,
		}
	}()

	return &Session{Messages: msgCh, Result: resCh}, nil
}

// qwenResultErrorText extracts the human-readable error from a failed result
// envelope. Qwen reports non-interactive failures (e.g. "No auth type is
// selected") under the top-level `error.message` while leaving the `result`
// field empty, so parse that nested object out of the raw line and fall back to
// resultText when it is absent — the caller must never surface an empty error.
func qwenResultErrorText(line, resultText string) string {
	var env struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &env); err == nil && env.Error != nil && env.Error.Message != "" {
		return env.Error.Message
	}
	return resultText
}

// qwenModel resolves the model name for usage attribution: the per-run override
// when set, otherwise the daemon default. Qwen always echoes the concrete model
// back in its assistant/result envelopes, so claudeResultUsage usually keys on
// that directly; this fallback only matters when usage arrives without a model.
func qwenModel(opts ExecOptions) string {
	if opts.Model != "" {
		return opts.Model
	}
	return defaultQwenModel
}

// ── Arg builder ──

// qwenBlockedArgs are flags hardcoded by the daemon that must not be overridden
// by user-configured custom_args. Mirrors geminiBlockedArgs since qwen-code
// shares gemini-cli's argument surface.
var qwenBlockedArgs = map[string]blockedArgMode{
	"-p":     blockedWithValue,  // non-interactive prompt
	"--yolo": blockedStandalone, // auto-approve tool use
	"-o":     blockedWithValue,  // stream-json output format
}

// buildQwenArgs assembles the argv for a one-shot qwen invocation.
//
// Flags (gemini-style; qwen-code is a gemini-cli fork):
//
//	-p <prompt>             non-interactive prompt (the user's task)
//	-o stream-json          streaming NDJSON output (already Claude schema)
//	--yolo                  auto-approve all tool executions
//	-m <model>              model (defaults to qwen3.7-max)
//	--auth-type openai      pick the OpenAI auth path headlessly (default)
//	--append-system-prompt  append developer/system instructions (if provided)
//	-r <session>            resume a previous session (if provided)
func buildQwenArgs(prompt string, opts ExecOptions, logger *slog.Logger) []string {
	args := []string{
		"-p", prompt,
		"-o", "stream-json",
		"--yolo",
		"-m", qwenModel(opts),
	}
	// qwen-code refuses to run non-interactively unless an auth type is
	// selected (it errors with "No auth type is selected" otherwise). The
	// mlamp gateway behind qwen3.7-max is OpenAI-only, so default to the
	// openai auth path — this keeps the provider self-sufficient without
	// requiring a pre-seeded ~/.qwen/settings.json. Users who run a different
	// auth mode (e.g. qwen-oauth) can override by passing their own
	// --auth-type in custom_args; we skip our default when they do so the CLI
	// never sees two conflicting values.
	if !customArgsHaveFlag(opts.CustomArgs, "--auth-type") {
		args = append(args, "--auth-type", "openai")
	}
	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
	}
	if opts.ResumeSessionID != "" {
		args = append(args, "-r", opts.ResumeSessionID)
	}
	args = append(args, filterCustomArgs(opts.CustomArgs, qwenBlockedArgs, logger)...)
	return args
}

// customArgsHaveFlag reports whether the user-supplied custom args already
// carry the given flag, in either `--flag value` or `--flag=value` form. Shell
// quoting is stripped first so a quoted `--auth-type='qwen-oauth'` is matched
// the same as the bare form (mirroring filterCustomArgs).
func customArgsHaveFlag(args []string, flag string) bool {
	for _, raw := range args {
		arg := unshellQuoteArg(raw)
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

// buildQwenEnv wraps buildEnv and defaults QWEN_CODE_SUPPRESS_YOLO_WARNING=1 so
// the CLI's interactive "YOLO mode is dangerous" warning doesn't pollute the
// stream when we run headless with --yolo. If the caller sets the key
// explicitly it wins, preserving the ability to re-enable the warning.
func buildQwenEnv(extra map[string]string) []string {
	const suppressKey = "QWEN_CODE_SUPPRESS_YOLO_WARNING"
	if _, ok := extra[suppressKey]; ok {
		return buildEnv(extra)
	}
	merged := make(map[string]string, len(extra)+1)
	for k, v := range extra {
		merged[k] = v
	}
	merged[suppressKey] = "1"
	return buildEnv(merged)
}
