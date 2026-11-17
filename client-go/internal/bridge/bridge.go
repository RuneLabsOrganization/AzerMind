// Package bridge connects the AzerMind addon transport to headless agent CLIs.
package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/agents"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/config"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/models"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/state"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/transport"
)

type running struct {
	job    protocol.Job
	cancel context.CancelFunc
}
type Client struct {
	Config     config.Config
	BridgeDir  string
	DefaultCWD string
	Publisher  *transport.Publisher
	State      *state.Store
	mu         sync.Mutex
	active     map[string]running
	queued     map[string]protocol.Job
	queueOrder []string
	work       sync.WaitGroup
	progress   map[string]time.Time
}

func New(c config.Config, bridgeDir, cwd string) (*Client, error) {
	s, err := state.Open(filepath.Join(bridgeDir, "state.json"))
	if err != nil {
		return nil, err
	}
	settings, codex := models.Snapshot(c)
	opts := protocol.LuaOptions{CWD: cwd, Agent: c.Agent, Agents: agents.Names, AIDefaults: protocol.Defaults{Agent: c.Agent, Model: c.Agents[c.Agent].Model, Reasoning: c.Agents[c.Agent].Reasoning}, AgentSettings: settings, CodexModels: codex}
	publisher := transport.NewPublisher(c.Transport(), opts)
	publisher.Presence = s.Presence
	return &Client{Config: c, BridgeDir: bridgeDir, DefaultCWD: cwd, State: s, Publisher: publisher, active: map[string]running{}, queued: map[string]protocol.Job{}, progress: map[string]time.Time{}}, nil
}
func (c *Client) Submit(j protocol.Job) {
	if j.Cancel > 0 {
		c.cancel(j)
		return
	}
	if j.Settings {
		c.settings(j)
		return
	}
	if c.State.SeenJob(j) {
		return
	}
	if j.ContextText != nil {
		if err := c.State.SetContext(j); err != nil {
			fmt.Fprintln(os.Stderr, "context:", err)
		}
	}
	if j.Forget {
		_ = c.State.Forget(j)
		_ = c.State.Mark(j)
		_ = c.Publisher.Signal("ack", j.ID, true)
		_ = c.Publisher.Publish(protocol.ChatKey(j), protocol.Reply{Chat: j.Chat, ID: j.ID, Status: "ack", Session: j.Session})
		return
	}
	if j.Hello {
		_ = c.State.Mark(j)
		_ = c.Publisher.Signal("ack", j.ID, true)
		_ = c.Publisher.PublishNow()
		return
	}
	key := protocol.ChatKey(j)
	c.mu.Lock()
	if r, exists := c.active[key]; exists && r.job.ID == j.ID {
		c.mu.Unlock()
		return
	}
	if q, exists := c.queued[key]; exists && q.ID == j.ID {
		c.mu.Unlock()
		return
	}
	if len(c.active) >= c.Config.MaxParallel || c.active[key].cancel != nil {
		if _, exists := c.queued[key]; !exists {
			c.queueOrder = append(c.queueOrder, key)
		}
		c.queued[key] = j
		c.mu.Unlock()
		return
	}
	c.startLocked(key, j)
	c.mu.Unlock()
}
func (c *Client) startLocked(key string, j protocol.Job) {
	ctx, cancel := context.WithCancel(context.Background())
	c.active[key] = running{job: j, cancel: cancel}
	c.work.Add(1)
	go func() { defer c.work.Done(); c.run(ctx, j); c.complete(key, j.ID) }()
}
func (c *Client) complete(key string, id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.active[key]; ok && r.job.ID == id {
		delete(c.active, key)
	}
	if len(c.active) >= c.Config.MaxParallel {
		return
	}
	remaining := c.queueOrder[:0]
	for _, k := range c.queueOrder {
		j, ok := c.queued[k]
		if !ok {
			continue
		}
		if len(c.active) < c.Config.MaxParallel {
			if _, busy := c.active[k]; !busy {
				delete(c.queued, k)
				c.startLocked(k, j)
				continue
			}
		}
		remaining = append(remaining, k)
	}
	c.queueOrder = remaining
}
func (c *Client) Wait() {
	for {
		c.work.Wait()
		c.mu.Lock()
		idle := len(c.active) == 0 && len(c.queued) == 0
		c.mu.Unlock()
		if idle {
			return
		}
	}
}
func (c *Client) cancel(j protocol.Job) {
	_ = c.State.Mark(protocol.Job{Session: j.Session, ID: j.Cancel})
	key := protocol.ChatKey(j)
	c.mu.Lock()
	if q, ok := c.queued[key]; ok && q.ID == j.Cancel {
		delete(c.queued, key)
	}
	if r, ok := c.active[key]; ok && r.job.ID == j.Cancel {
		r.cancel()
	}
	c.mu.Unlock()
	_ = c.Publisher.Publish("control:"+key, protocol.Reply{Chat: j.Chat, ID: j.ID, Status: "cancel-ack", Session: j.Session})
}
func (c *Client) settings(j protocol.Job) {
	var request struct {
		Action    string `json:"action"`
		Agent     string `json:"agent"`
		Model     string `json:"model"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(j.Text), &request); err != nil {
		c.settingReply(j, "settings-error", "Invalid settings JSON.")
		return
	}
	if request.Action == "refresh-models" {
		go func() {
			settings, codex := models.Refresh(context.Background(), c.Config)
			c.Publisher.Mu.Lock()
			c.Publisher.Opts.AgentSettings = settings
			c.Publisher.Opts.CodexModels = codex
			c.Publisher.Mu.Unlock()
			c.settingReply(j, "models-refreshed", "Model choices refreshed.")
		}()
		return
	}
	settings, _ := models.Snapshot(c.Config)
	if err := models.Validate(request.Agent, request.Model, request.Reasoning, settings); err != nil {
		c.settingReply(j, "settings-error", err.Error())
		return
	}
	file := filepath.Join(c.BridgeDir, "config.json")
	b, err := os.ReadFile(file)
	if err != nil {
		c.settingReply(j, "settings-error", err.Error())
		return
	}
	var raw map[string]interface{}
	if err = json.Unmarshal(b, &raw); err != nil {
		c.settingReply(j, "settings-error", err.Error())
		return
	}
	existing, _ := raw["agents"].(map[string]interface{})
	if existing == nil {
		existing = map[string]interface{}{}
	}
	own, _ := existing[request.Agent].(map[string]interface{})
	if own == nil {
		own = map[string]interface{}{}
	}
	own["model"] = request.Model
	own["reasoning"] = request.Reasoning
	existing[request.Agent] = own
	raw["agent"] = request.Agent
	raw["agents"] = existing
	body, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		c.settingReply(j, "settings-error", err.Error())
		return
	}
	if err = transport.AtomicWrite(file, append(body, '\n')); err != nil {
		c.settingReply(j, "settings-error", err.Error())
		return
	}
	c.mu.Lock()
	c.Config.Agent = request.Agent
	ac := c.Config.Agents[request.Agent]
	ac.Model = request.Model
	ac.Reasoning = request.Reasoning
	c.Config.Agents[request.Agent] = ac
	c.mu.Unlock()
	settings, _ = models.Snapshot(c.Config)
	c.Publisher.Mu.Lock()
	c.Publisher.Opts.Agent = request.Agent
	c.Publisher.Opts.AgentSettings = settings
	c.Publisher.Opts.AIDefaults = protocol.Defaults{Agent: request.Agent, Model: request.Model, Reasoning: request.Reasoning}
	c.Publisher.Mu.Unlock()
	c.settingReply(j, "settings-saved", "Default settings saved.")
}
func (c *Client) settingReply(j protocol.Job, status, text string) {
	_ = c.State.Mark(j)
	_ = c.Publisher.Publish("settings:"+protocol.ChatKey(j), protocol.Reply{Chat: j.Chat, ID: j.ID, Status: status, Text: text, Session: j.Session})
}
func (c *Client) publish(j protocol.Job, status, text, session, agent string, denied []string, urgent bool) {
	key := protocol.ChatKey(j)
	now := time.Now()
	c.mu.Lock()
	if !urgent && now.Sub(c.progress[key]) < time.Duration(c.Config.ProgressWriteMs)*time.Millisecond {
		c.mu.Unlock()
		return
	}
	c.progress[key] = now
	c.mu.Unlock()
	summary := ""
	if status == "done" {
		summary = splitSummary(text)
	}
	_ = c.Publisher.Publish(key, protocol.Reply{Chat: j.Chat, ID: j.ID, Status: status, Text: text, CWD: j.CWD, Session: session, Agent: agent, Summary: summary, Denied: denied})
}
func splitSummary(text string) string {
	index := -1
	span := 0
	offset := 0
	for _, line := range strings.SplitAfter(text, "\n") {
		l := strings.TrimSpace(line)
		for _, prefix := range []string{"TL;DR:", "**TL;DR:**", "## TL;DR", "tldr:"} {
			if strings.HasPrefix(strings.ToLower(l), strings.ToLower(prefix)) {
				index = offset
				span = len(line) - len(strings.TrimLeft(line, " \t")) + len(prefix)
				break
			}
		}
		offset += len(line)
	}
	if index < 0 {
		return ""
	}
	return strings.TrimSpace(text[index+span:])
}
func systemPrompt(ctx, primer string) string {
	prompt := "The user is talking from inside World of Warcraft through the azermind addon. Replies appear as plain text. Keep them compact. End EVERY reply with a final block starting with \"TL;DR:\" on its own line, followed by one or two short summary lines. Put nothing after it."
	if ctx == "" {
		return prompt
	}
	prompt += "\n\nTheir in-game situation when the message was written, as reported by the addon:\n" + ctx + "\n\nUse this when relevant; linked item, spell and quest tooltips are included in the context."
	if primer != "" {
		prompt += "\n\nReference for writing addons and macros for this client:\n" + primer
	}
	return prompt
}
func (c *Client) run(ctx context.Context, j protocol.Job) {
	j.CWD = protocol.ResolveCWD(j.CWD, c.DefaultCWD)
	_ = c.Publisher.Signal("sig", j.ID, false)
	_ = c.Publisher.ResetBeats(j.ID)
	_ = c.Publisher.Signal("ack", j.ID, true)
	finish := func(status, text, session, agent string, denied []string) {
		_ = c.State.Mark(j)
		c.publish(j, status, text, session, agent, denied, true)
		_ = c.Publisher.Signal("sig", j.ID, true)
	}
	if info, err := os.Stat(j.CWD); err != nil || !info.IsDir() {
		finish("error", fmt.Sprintf("Folder does not exist: %s. Use /azermind cd <folder>.", j.CWD), "", j.Agent, nil)
		return
	}
	agent := j.Agent
	if agent == "" {
		agent = c.Config.Agent
	}
	agent = strings.ToLower(agent)
	if !agents.IsAgent(agent) {
		finish("error", "Unknown agent: "+agent, "", agent, nil)
		return
	}
	cfg := c.Config.Agents[agent]
	if j.Model != "" {
		cfg.Model = j.Model
	}
	if j.Reasoning != "" {
		cfg.Reasoning = j.Reasoning
	}
	settings := cfg.Model + ":" + cfg.Reasoning
	resume, err := c.State.GetSession(j, agent, j.CWD, settings)
	if err != nil {
		finish("error", "State error: "+err.Error(), "", agent, nil)
		return
	}
	ctxText := ""
	if !(c.Config.GameContext != nil && !*c.Config.GameContext) {
		ctxText = c.State.ContextText()
	}
	primer := ""
	if c.Config.PrimerFile != "" && ctxText != "" {
		p := c.Config.PrimerFile
		if !filepath.IsAbs(p) {
			p = filepath.Join(filepath.Dir(c.BridgeDir), p)
		}
		if b, err := os.ReadFile(p); err == nil {
			primer = string(b)
		}
	}
	prompt := systemPrompt(ctxText, primer)
	request := agents.Request{Agent: agent, CWD: j.CWD, Prompt: j.Text, System: prompt, SystemShort: systemPrompt(ctxText, ""), Resume: resume, Timeout: time.Duration(c.Config.TimeoutMs) * time.Millisecond, Config: cfg}
	_ = c.Publisher.Publish(protocol.ChatKey(j), protocol.Reply{Chat: j.Chat, ID: j.ID, Status: "working", Text: "thinking...", CWD: j.CWD, Session: resume, Agent: agent})
	var last []string
	result, err := agents.Run(ctx, request, func(line string) {
		last = append(last, line)
		if len(last) > 10 {
			last = last[len(last)-10:]
		}
		_ = c.Publisher.Beat(j.ID)
		c.publish(j, "working", strings.Join(last, "\n"), resume, agent, nil, false)
	})
	if ctx.Err() == context.Canceled {
		return
	}
	if err != nil {
		finish("error", err.Error(), resume, agent, nil)
		return
	}
	if result.Session != "" {
		_ = c.State.SaveSession(j, result.Session, agent, j.CWD)
	}
	text := strings.TrimSpace(result.Text)
	if len(result.Notes) > 0 {
		text += "\n\n[bridge] " + strings.Join(result.Notes, "\n\n[bridge] ")
	}
	status := "done"
	if result.Error {
		status = "error"
	}
	finish(status, text, result.Session, agent, result.Denied)
}

func (c *Client) PollSavedVariables() {
	file := c.Config.SavedVariablesFile
	if file == "" {
		return
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	if job, ok := protocol.ParseOutbox(string(b)); ok {
		c.Submit(job)
	}
}
func (c *Client) RunCapture(ctx context.Context) {
	cap := c.Config.Capture
	if cap.Enabled != nil && !*cap.Enabled {
		return
	}
	assembler := protocol.NewAssembler()
	for ctx.Err() == nil {
		var program string
		var args []string
		bridgeDir := c.BridgeDir
		common := []string{"--cell", strconv.Itoa(cap.CellPx), "--cells", strconv.Itoa(cap.CellsPerRow), "--max-rows", strconv.Itoa(cap.MaxRows), "--interval-ms", strconv.Itoa(cap.IntervalMs), "--process-name", cap.ProcessName}
		switch runtime.GOOS {
		case "windows":
			program = "powershell.exe"
			args = []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filepath.Join(bridgeDir, "capture.ps1"), "-Cell", strconv.Itoa(cap.CellPx), "-Cells", strconv.Itoa(cap.CellsPerRow), "-MaxRows", strconv.Itoa(cap.MaxRows), "-IntervalMs", strconv.Itoa(cap.IntervalMs), "-ProcessName", cap.ProcessName, "-ClientFolder", filepath.Base(filepath.Dir(filepath.Dir(c.Config.AddonDir)))}
		case "darwin":
			program = cap.Python
			args = append([]string{filepath.Join(bridgeDir, "capture_mac.py")}, common...)
			if cap.WindowName != "" {
				args = append(args, "--window-name", cap.WindowName)
			}
		case "linux":
			program = cap.Python
			args = append([]string{filepath.Join(bridgeDir, "capture_x11.py")}, common...)
			if cap.WindowName != "" {
				args = append(args, "--window-name", cap.WindowName)
			}
			if cap.KeepComposited {
				args = append(args, "--keep-composited")
			}
		default:
			return
		}
		cmd := exec.CommandContext(ctx, program, args...)
		stdout, err := cmd.StdoutPipe()
		if err == nil {
			err = cmd.Start()
		}
		if err == nil {
			scan := bufio.NewScanner(stdout)
			scan.Buffer(make([]byte, 16384), 131072)
			for scan.Scan() {
				var ev struct {
					ID    *int   `json:"id"`
					Text  string `json:"text"`
					Info  string `json:"info"`
					Warn  string `json:"warn"`
					Error string `json:"error"`
				}
				if json.Unmarshal(scan.Bytes(), &ev) != nil {
					continue
				}
				if ev.ID == nil {
					continue
				}
				if payload, ok := assembler.Add(ev.Text); ok {
					for _, job := range protocol.JobsFromStrip(*ev.ID, payload) {
						c.Submit(job)
					}
				}
			}
			err = cmd.Wait()
		}
		if ctx.Err() != nil {
			return
		}
		fmt.Fprintln(os.Stderr, "capture stopped:", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
func (c *Client) Run(ctx context.Context) {
	c.PollSavedVariables()
	presence := time.Duration(c.Config.PresenceIntervalMs) * time.Millisecond
	poll := time.Duration(c.Config.PollMs) * time.Millisecond
	go c.RunCapture(ctx)
	tick := time.NewTicker(poll)
	defer tick.Stop()
	beats := time.NewTicker(presence)
	defer beats.Stop()
	_ = c.Publisher.PresenceBeat()
	_ = c.State.SetPresence(c.Publisher.Presence)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.PollSavedVariables()
		case <-beats.C:
			_ = c.Publisher.PresenceBeat()
			_ = c.State.SetPresence(c.Publisher.Presence)
		}
	}
}
