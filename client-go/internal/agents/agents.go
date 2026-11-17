// Package agents manages headless CLIs used by AzerMind. The CLI contract is
// mirrored from bridge/agents.js; agents are not bundled with the client.
package agents

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
)

var Names = []string{"claude", "codex", "grok", "agy", "hermes"}

func IsAgent(id string) bool {
	for _, x := range Names {
		if id == x {
			return true
		}
	}
	return false
}

type Config struct {
	Models             []protocol.Model `json:"models"`
	Path               string           `json:"path"`
	Model              string           `json:"model"`
	Reasoning          string           `json:"reasoning"`
	PermissionMode     string           `json:"permissionMode"`
	NetworkAccess      bool             `json:"networkAccess"`
	ReasoningSupported bool             `json:"reasoningSupported"`
	AllowedTools       []string         `json:"allowedTools"`
	DeniedTools        []string         `json:"deniedTools"`
	ExtraArgs          []string         `json:"extraArgs"`
}
type Request struct {
	Agent       string
	CWD         string
	Prompt      string
	System      string
	SystemShort string
	Resume      string
	Timeout     time.Duration
	Config      Config
}
type Invocation struct {
	Command    string
	Args       []string
	Stdin      string
	PromptFile string
	Env        map[string]string
	StreamText bool
}

func appendArgs(a []string, extra []string) []string { return append(a, extra...) }
func contextBlock(s string) string {
	return "[Context from the AzerMind bridge, not written by the user]\n" + s + "\n[End of context]\n\n"
}
func grokRules(s string) []string {
	m := regexp.MustCompile(`^Bash\(([^\s:()]+):\*\)$`).FindStringSubmatch(s)
	if len(m) > 0 {
		return []string{"Bash(" + m[1] + " *)", "Bash(" + m[1] + ")"}
	}
	if s == "" {
		return nil
	}
	return []string{s}
}

func Build(r Request) (Invocation, error) {
	id := strings.ToLower(strings.TrimSpace(r.Agent))
	if !IsAgent(id) {
		return Invocation{}, fmt.Errorf("unknown agent %q", r.Agent)
	}
	c := r.Config
	mode := c.PermissionMode
	if mode == "" {
		mode = "acceptEdits"
	}
	command := c.Path
	if command == "" {
		command = id
	}
	inv := Invocation{Command: command}
	switch id {
	case "claude":
		inv.Args = []string{"-p", "--output-format", "stream-json", "--verbose", "--permission-mode", mode}
		if len(c.AllowedTools) > 0 {
			inv.Args = append(inv.Args, "--allowedTools")
			inv.Args = append(inv.Args, c.AllowedTools...)
		}
		if len(c.DeniedTools) > 0 {
			inv.Args = append(inv.Args, "--disallowedTools")
			inv.Args = append(inv.Args, c.DeniedTools...)
		}
		if c.Model != "" {
			inv.Args = append(inv.Args, "--model", c.Model)
		}
		if c.Reasoning != "" {
			inv.Args = append(inv.Args, "--effort", c.Reasoning)
		}
		if r.Resume != "" {
			inv.Args = append(inv.Args, "--resume", r.Resume)
		}
		if r.System != "" {
			inv.Args = append(inv.Args, "--append-system-prompt", r.System)
		}
		inv.Args = appendArgs(inv.Args, c.ExtraArgs)
		inv.Stdin = r.Prompt
		inv.Env = map[string]string{"CLAUDECODE": ""}
	case "codex":
		if c.NetworkAccess {
			inv.Args = append(inv.Args, "-c", "sandbox_workspace_write.network_access=true")
		}
		inv.Args = append(inv.Args, "exec", "--json", "--skip-git-repo-check", "-C", r.CWD)
		if mode == "bypassPermissions" {
			inv.Args = append(inv.Args, "--dangerously-bypass-approvals-and-sandbox")
		} else if mode == "default" {
			inv.Args = append(inv.Args, "--sandbox", "read-only")
		} else {
			inv.Args = append(inv.Args, "--sandbox", "workspace-write")
		}
		if c.Model != "" {
			inv.Args = append(inv.Args, "-m", c.Model)
		}
		inv.Args = appendArgs(inv.Args, c.ExtraArgs)
		if c.Reasoning != "" {
			v, _ := json.Marshal(c.Reasoning)
			inv.Args = append(inv.Args, "-c", "model_reasoning_effort="+string(v))
		}
		if r.Resume != "" {
			inv.Args = append(inv.Args, "resume", r.Resume)
		}
		inv.Args = append(inv.Args, "-")
		ctx := r.System
		if r.Resume != "" {
			ctx = r.SystemShort
		}
		if ctx != "" {
			inv.Stdin = contextBlock(ctx)
		}
		inv.Stdin += r.Prompt
	case "grok":
		inv.PromptFile = r.Prompt
		inv.Args = []string{"--no-auto-update", "--output-format", "streaming-json", "--cwd", r.CWD, "--prompt-file", "{prompt-file}"}
		if mode == "bypassPermissions" {
			inv.Args = append(inv.Args, "--always-approve")
		} else {
			inv.Args = append(inv.Args, "--permission-mode", "dontAsk")
			allowed := []string{}
			if mode == "acceptEdits" {
				allowed = append(allowed, "Edit", "Read", "Grep")
			}
			for _, rule := range c.AllowedTools {
				allowed = append(allowed, grokRules(rule)...)
			}
			seen := map[string]bool{}
			for _, rule := range allowed {
				if !seen[rule] {
					seen[rule] = true
					inv.Args = append(inv.Args, "--allow", rule)
				}
			}
		}
		seen := map[string]bool{}
		for _, d := range c.DeniedTools {
			for _, rule := range grokRules(d) {
				if !seen[rule] {
					seen[rule] = true
					inv.Args = append(inv.Args, "--deny", rule)
				}
			}
		}
		if c.Model != "" {
			inv.Args = append(inv.Args, "-m", c.Model)
		}
		if c.Reasoning != "" && c.ReasoningSupported {
			inv.Args = append(inv.Args, "--effort", c.Reasoning)
		}
		if r.Resume != "" {
			inv.Args = append(inv.Args, "-r", r.Resume)
		}
		if r.System != "" {
			inv.Args = append(inv.Args, "--append-system-prompt", r.System)
		}
		inv.Args = appendArgs(inv.Args, c.ExtraArgs)
		inv.Env = map[string]string{"GROK_DISABLE_AUTOUPDATER": "1"}
	case "agy":
		ctx := r.System
		if r.Resume != "" {
			ctx = r.SystemShort
		}
		text := contextBlock(ctx)
		if len(text) > 12000 {
			text = text[:12000]
		}
		available := 24000 - len(text) - len("\n[User prompt truncated to fit the agy command line.]")
		if len(r.Prompt) > available {
			inv.PromptFile = ""
			text += r.Prompt[:available] + "\n[User prompt truncated to fit the agy command line.]"
		} else {
			text += r.Prompt
		}
		timeout := r.Timeout
		if timeout == 0 {
			timeout = 30 * time.Minute
		}
		inv.Args = []string{"-p=" + text, "--output-format", "stream-json", "--add-dir", r.CWD, "--print-timeout", fmt.Sprintf("%ds", int(timeout.Seconds()+0.999))}
		switch mode {
		case "default":
			inv.Args = append(inv.Args, "--mode", "plan")
		case "bypassPermissions":
			inv.Args = append(inv.Args, "--dangerously-skip-permissions", "--disable-slash-commands")
		default:
			inv.Args = append(inv.Args, "--mode", "accept-edits", "--disable-slash-commands")
		}
		if r.Resume != "" {
			inv.Args = append(inv.Args, "--conversation", r.Resume)
		}
		if c.Model != "" {
			inv.Args = append(inv.Args, "--model", c.Model)
		}
		if c.Reasoning != "" {
			inv.Args = append(inv.Args, "--effort", c.Reasoning)
		}
		inv.Args = appendArgs(inv.Args, c.ExtraArgs)
	case "hermes":
		inv.StreamText = true
		inv.Args = []string{"chat", "--query-file", "-", "-Q", "--in", r.CWD, "--source", "tool"}
		if r.Resume != "" {
			inv.Args = append(inv.Args, "--resume", r.Resume)
		}
		if c.Model != "" {
			inv.Args = append(inv.Args, "-m", c.Model)
		}
		if c.Reasoning != "" {
			inv.Args = append(inv.Args, "--reasoning", c.Reasoning)
		}
		for _, a := range c.ExtraArgs {
			if !regexp.MustCompile(`^(?:-y|--yolo)(?:=.*)?$`).MatchString(a) {
				inv.Args = append(inv.Args, a)
			}
		}
		ctx := r.System
		if r.Resume != "" {
			ctx = r.SystemShort
		}
		if ctx != "" {
			inv.Stdin = contextBlock(ctx)
		}
		inv.Stdin += r.Prompt
	}
	return inv, nil
}

type Event struct {
	Session  string
	Progress []string
	Denied   []string
	Notes    []string
	Done     *Result
}
type Result struct {
	Text    string
	Error   bool
	Session string
	Notes   []string
	Denied  []string
}
type Parser struct {
	Agent    string
	last     string
	text     string
	thought  string
	response string
	calls    map[string]string
}

func NewParser(id string) *Parser { return &Parser{Agent: id, calls: map[string]string{}} }
func obj(x interface{}) map[string]interface{} {
	v, _ := x.(map[string]interface{})
	if v == nil {
		return map[string]interface{}{}
	}
	return v
}
func str(v interface{}) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}
func child(m map[string]interface{}, k string) map[string]interface{} { return obj(m[k]) }
func snippet(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > 140 {
		return string([]rune(s)[:140]) + "..."
	}
	return s
}
func firstLine(s string) string {
	s = strings.Split(s, "\n")[0]
	return string([]rune(s)[:min(len([]rune(s)), 110)])
}
func base(s string) string { return filepath.Base(strings.ReplaceAll(s, "\\", "/")) }
func (p *Parser) Feed(line []byte) (Event, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, err
	}
	e := Event{}
	switch p.Agent {
	case "claude":
		if s := str(raw["session_id"]); s != "" {
			e.Session = s
		}
		if str(raw["type"]) == "assistant" {
			blocks, _ := child(raw, "message")["content"].([]interface{})
			for _, b := range blocks {
				v := obj(b)
				if str(v["type"]) == "tool_use" {
					e.Progress = append(e.Progress, str(v["name"]))
				} else if str(v["type"]) == "text" && str(v["text"]) != "" {
					e.Progress = append(e.Progress, snippet(str(v["text"])))
				}
			}
		}
		if str(raw["type"]) == "result" {
			text := str(raw["result"])
			if val, ok := raw["result"].(string); ok {
				text = val
			} else if raw["result"] != nil {
				j, _ := json.MarshalIndent(raw["result"], "", "  ")
				text = string(j)
			}
			e.Done = &Result{Text: text, Error: raw["is_error"] == true}
			for _, v := range asList(raw["permission_denials"]) {
				d := obj(v)
				rule := str(d["tool_name"])
				if rule == "Bash" {
					cmd := strings.Fields(str(child(d, "tool_input")["command"]))
					if len(cmd) > 0 {
						rule = "Bash(" + cmd[0] + ":*)"
					}
				}
				if rule != "" {
					e.Denied = append(e.Denied, rule)
				}
			}
		}
	case "codex":
		typ := str(raw["type"])
		item := child(raw, "item")
		if typ == "thread.started" {
			e.Session = str(raw["thread_id"])
		}
		if typ == "item.started" || typ == "item.completed" {
			switch str(item["type"]) {
			case "command_execution":
				e.Progress = append(e.Progress, "$ "+firstLine(str(item["command"])))
			case "file_change":
				e.Progress = append(e.Progress, "edit files")
			case "web_search":
				e.Progress = append(e.Progress, "search: "+str(item["query"]))
			case "mcp_tool_call":
				e.Progress = append(e.Progress, "tool: "+str(item["server"])+"."+str(item["tool"]))
			case "agent_message":
				if typ == "item.completed" {
					p.last = str(item["text"])
					e.Progress = append(e.Progress, snippet(p.last))
				}
			case "reasoning":
				if typ == "item.completed" {
					e.Progress = append(e.Progress, "~ "+snippet(str(item["text"])))
				}
			}
		}
		if typ == "turn.completed" {
			e.Done = &Result{Text: p.last}
		}
		if typ == "turn.failed" {
			e.Done = &Result{Text: str(child(raw, "error")["message"]), Error: true}
		}
		if typ == "error" {
			e.Done = &Result{Text: str(raw["message"]), Error: true}
		}
	case "grok":
		typ := str(raw["type"])
		chunk := str(raw["data"])
		if chunk == "" {
			chunk = str(child(raw, "data")["text"])
		}
		if chunk == "" {
			chunk = str(raw["text"])
		}
		switch typ {
		case "text":
			p.text += chunk
		case "thought":
			p.thought += chunk
		case "tool_call":
			if p.text != "" {
				p.last = p.text
				e.Progress = append(e.Progress, snippet(p.text))
				p.text = ""
			}
			name := str(raw["toolName"])
			if name == "" {
				name = str(raw["title"])
			}
			e.Progress = append(e.Progress, name)
			p.calls[str(raw["toolCallId"])] = name
		case "tool_call_update":
			if strings.Contains(strings.ToLower(str(raw["status"])), "denied") {
				if name := p.calls[str(raw["toolCallId"])]; name != "" {
					e.Notes = append(e.Notes, "Grok denied tool: "+name)
				}
			}
		case "end":
			if p.thought != "" {
				e.Progress = append(e.Progress, "~ "+snippet(p.thought))
			}
			text := p.last
			if strings.TrimSpace(p.text) != "" {
				text = p.text
			}
			e.Session = str(raw["sessionId"])
			if e.Session == "" {
				e.Session = str(raw["session_id"])
			}
			e.Done = &Result{Text: strings.TrimSpace(text)}
		case "error":
			e.Done = &Result{Text: str(raw["message"]), Error: true}
		}
	case "agy":
		if str(raw["event"]) == "init" {
			e.Session = str(raw["conversation_id"])
		}
		if str(raw["event"]) == "step_update" {
			v := child(raw, "step_update")
			if str(v["step_type"]) == "tool" && str(v["state"]) == "ACTIVE" {
				e.Progress = append(e.Progress, str(v["tool_name"]))
			}
			if str(v["step_type"]) == "agent_response" {
				p.response += str(v["text_delta"])
			}
		}
		if str(raw["event"]) == "result" {
			v := child(raw, "result")
			e.Session = str(v["conversation_id"])
			out := str(v["response"])
			if out == "" {
				out = p.response
			}
			e.Done = &Result{Text: out, Error: str(v["status"]) != "SUCCESS"}
		}
	}
	return e, nil
}
func asList(v interface{}) []interface{} { x, _ := v.([]interface{}); return x }

var hermesSession = regexp.MustCompile(`(?i)session_id:\s*(\S+)`)

func (p *Parser) Finish(stdout, stderr string, code int) Event {
	e := Event{}
	if p.Agent != "hermes" {
		return e
	}
	s := hermesSession.FindStringSubmatch(stderr)
	if len(s) > 0 {
		e.Session = s[1]
	}
	text := strings.TrimSpace(stdout)
	if text == "" && code != 0 {
		text = stderr
	}
	e.Done = &Result{Text: text, Error: code != 0}
	return e
}

// Run executes a single turn and streams progress while preserving the CLI session ID.
func Run(ctx context.Context, r Request, onProgress func(string)) (Result, error) {
	inv, err := Build(r)
	if err != nil {
		return Result{}, err
	}
	cmdPath := inv.Command
	if _, err := exec.LookPath(cmdPath); err != nil {
		return Result{}, fmt.Errorf("%s CLI not found (%s): %w", r.Agent, cmdPath, err)
	}
	var promptFile string
	if inv.PromptFile != "" {
		f, err := os.CreateTemp("", "azermind-prompt-*.txt")
		if err != nil {
			return Result{}, err
		}
		promptFile = f.Name()
		defer os.Remove(promptFile)
		if _, err = f.WriteString(inv.PromptFile); err != nil {
			f.Close()
			return Result{}, err
		}
		if err = f.Close(); err != nil {
			return Result{}, err
		}
		for i, a := range inv.Args {
			if a == "{prompt-file}" {
				inv.Args[i] = promptFile
			}
		}
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdPath, inv.Args...)
	cmd.Dir = r.CWD
	env := os.Environ()
	for k, v := range inv.Env {
		if v == "" {
			filtered := env[:0]
			for _, pair := range env {
				if !strings.HasPrefix(pair, k+"=") {
					filtered = append(filtered, pair)
				}
			}
			env = filtered
		} else {
			env = append(env, k+"="+v)
		}
	}
	cmd.Env = env
	if inv.Stdin != "" {
		cmd.Stdin = strings.NewReader(inv.Stdin)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return Result{}, err
	}
	parser := NewParser(r.Agent)
	var result *Result
	session := r.Resume
	var notes, denied []string
	if inv.StreamText {
		out := new(bytes.Buffer)
		_, err = out.ReadFrom(stdout)
		exit := cmd.Wait()
		code := 0
		if exit != nil {
			code = 1
		}
		e := parser.Finish(out.String(), stderr.String(), code)
		result = e.Done
		if e.Session != "" {
			session = e.Session
		}
	} else {
		scan := bufio.NewScanner(stdout)
		scan.Buffer(make([]byte, 65536), 10*1024*1024)
		for scan.Scan() {
			e, parseErr := parser.Feed(scan.Bytes())
			if parseErr != nil {
				continue
			}
			if e.Session != "" {
				session = e.Session
			}
			for _, s := range e.Progress {
				if onProgress != nil && s != "" {
					onProgress(s)
				}
			}
			notes = append(notes, e.Notes...)
			denied = append(denied, e.Denied...)
			if e.Done != nil {
				result = e.Done
			}
		}
		err = scan.Err()
		exit := cmd.Wait()
		if err != nil {
			return Result{}, err
		}
		if exit != nil && result == nil {
			return Result{}, fmt.Errorf("%s exited: %w (%s)", r.Agent, exit, stderr.String())
		}
	}
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if result == nil {
		return Result{}, errors.New("agent returned no result")
	}
	result.Session = session
	result.Notes = notes
	result.Denied = denied
	return *result, nil
}
