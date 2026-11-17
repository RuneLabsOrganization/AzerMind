// Package protocol implements the AzerMind addon/bridge wire format.
// Keep changes compatible with bridge/protocol.js: the Lua addon is a separate release.
package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const FieldSep = "\x1f"
const RecordSep = "\x1e"
const ContextLimit = 14000
const ContextPartial = "[Game context partial: truncated by bridge at 14000 UTF-8 bytes.]"

var digit = regexp.MustCompile(`^\d+$`)
var partSession = regexp.MustCompile(`^[a-zA-Z0-9]{1,128}$`)
var fragmentHex = regexp.MustCompile(`^[a-fA-F0-9]+$`)

type Job struct {
	Session     string   `json:"session"`
	Chat        string   `json:"chat"`
	ID          int      `json:"id"`
	CWD         string   `json:"cwd"`
	Name        string   `json:"name,omitempty"`
	Text        string   `json:"text"`
	Via         string   `json:"via"`
	ContextText *string  `json:"ctx,omitempty"`
	NewSession  bool     `json:"newSession,omitempty"`
	Hello       bool     `json:"hello,omitempty"`
	Forget      bool     `json:"forget,omitempty"`
	HasContext  bool     `json:"context,omitempty"`
	Settings    bool     `json:"settings,omitempty"`
	Cancel      int      `json:"cancel,omitempty"`
	Allow       []string `json:"allow,omitempty"`
	Agent       string   `json:"agent,omitempty"`
	Model       string   `json:"model,omitempty"`
	Reasoning   string   `json:"reasoning,omitempty"`
}

func ChatKey(job Job) string {
	chat := job.Chat
	if chat == "" {
		chat = "default"
	}
	return job.Session + ":" + chat
}
func SessionKey(job Job) string {
	if job.Chat != "" {
		return "chat:" + job.Chat
	}
	return ChatKey(job)
}
func SlotNumber(id, slots int) int {
	if slots <= 0 || id <= 0 {
		return 0
	}
	return (id-1)%slots + 1
}
func Pad3(n int) string { return fmt.Sprintf("%03d", n) }

func ParseFlags(raw string) Job {
	j := Job{}
	for _, tok := range strings.Split(raw, ";") {
		switch tok {
		case "n":
			j.NewSession = true
		case "h":
			j.Hello = true
		case "d":
			j.Forget = true
		case "c":
			j.HasContext = true
		case "settings":
			j.Settings = true
		default:
			switch {
			case strings.HasPrefix(tok, "cancel="):
				if n, e := strconv.Atoi(strings.TrimPrefix(tok, "cancel=")); e == nil && n >= 0 {
					j.Cancel = n
				}
			case strings.HasPrefix(tok, "allow="):
				for _, rule := range strings.Split(strings.TrimPrefix(tok, "allow="), ",") {
					if r := strings.TrimSpace(rule); r != "" {
						j.Allow = append(j.Allow, r)
					}
				}
			case strings.HasPrefix(tok, "agent="):
				j.Agent = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tok, "agent=")))
			case strings.HasPrefix(tok, "model="):
				j.Model = strings.TrimPrefix(tok, "model=")
			case strings.HasPrefix(tok, "reasoning="):
				j.Reasoning = strings.TrimPrefix(tok, "reasoning=")
			}
		}
	}
	return j
}

func JobsFromStrip(headerID int, payload string) []Job {
	var jobs []Job
	for _, rec := range strings.Split(payload, RecordSep) {
		p := strings.Split(rec, FieldSep)
		switch {
		case len(p) >= 7 && digit.MatchString(p[2]):
			id, _ := strconv.Atoi(p[2])
			j := ParseFlags(p[4])
			j.Session = p[0]
			j.Chat = p[1]
			j.ID = id
			j.CWD = p[3]
			j.Name = p[5]
			j.Via = "pixel"
			if j.HasContext && len(p) >= 8 {
				ctx := p[6]
				j.ContextText = &ctx
				j.Text = strings.Join(p[7:], FieldSep)
			} else {
				j.Text = strings.Join(p[6:], FieldSep)
			}
			jobs = append(jobs, j)
		case len(p) == 6 && digit.MatchString(p[2]):
			id, _ := strconv.Atoi(p[2])
			j := ParseFlags(p[4])
			j.Session = p[0]
			j.Chat = p[1]
			j.ID = id
			j.CWD = p[3]
			j.Text = p[5]
			j.Via = "pixel"
			jobs = append(jobs, j)
		case len(p) == 4:
			j := ParseFlags(p[2])
			j.Session = p[0]
			j.ID = headerID
			j.CWD = p[1]
			j.Text = p[3]
			j.Via = "pixel"
			jobs = append(jobs, j)
		}
	}
	return jobs
}

type assembly struct {
	count int
	parts map[int]string
	at    time.Time
}
type Assembler struct {
	mu      sync.Mutex
	pending map[string]*assembly
}

func NewAssembler() *Assembler                         { return &Assembler{pending: make(map[string]*assembly)} }
func (a *Assembler) Add(payload string) (string, bool) { return a.AddAt(payload, time.Now()) }

// AddAt returns complete payloads only. Non-fragment legacy payloads pass unchanged.
func (a *Assembler) AddAt(payload string, now time.Time) (string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pending == nil {
		a.pending = make(map[string]*assembly)
	}
	for k, v := range a.pending {
		if now.Sub(v.at) > 2*time.Minute {
			delete(a.pending, k)
		}
	}
	if !strings.HasPrefix(payload, "F1"+FieldSep) {
		return payload, true
	}
	p := strings.Split(payload, FieldSep)
	if len(p) != 6 || !partSession.MatchString(p[1]) || !digit.MatchString(p[2]) || !fragmentHex.MatchString(p[5]) || len(p[5]) < 2 || len(p[5]) > 5000 || len(p[5])%2 != 0 {
		return "", false
	}
	part, e1 := strconv.Atoi(p[3])
	count, e2 := strconv.Atoi(p[4])
	if e1 != nil || e2 != nil || count < 2 || count > 80 || part < 1 || part > count {
		return "", false
	}
	key := p[1] + ":" + p[2]
	v, exists := a.pending[key]
	if !exists {
		if len(a.pending) >= 32 {
			var oldest string
			var when time.Time
			for k, x := range a.pending {
				if oldest == "" || x.at.Before(when) {
					oldest = k
					when = x.at
				}
			}
			delete(a.pending, oldest)
		}
		v = &assembly{count: count, parts: make(map[int]string), at: now}
		a.pending[key] = v
	}
	if v.count != count {
		return "", false
	}
	if old, ok := v.parts[part]; ok && old != p[5] {
		return "", false
	}
	v.parts[part] = p[5]
	if len(v.parts) != count {
		return "", false
	}
	var b strings.Builder
	for i := 1; i <= count; i++ {
		partHex, ok := v.parts[i]
		if !ok {
			return "", false
		}
		raw, e := hex.DecodeString(partHex)
		if e != nil {
			return "", false
		}
		b.Write(raw)
		if b.Len() > 65000 {
			delete(a.pending, key)
			return "", false
		}
	}
	delete(a.pending, key)
	value := b.String()
	fields := strings.Split(value, FieldSep)
	if !utf8.ValidString(value) || len(fields) < 3 || fields[0] != p[1] || fields[2] != p[2] {
		return "", false
	}
	return value, true
}

var outboxBlock = regexp.MustCompile(`\["outbox"\]\s*=\s*\{([^}]*)\}`)

func fieldQuoted(block, key string) (string, bool) {
	m := regexp.MustCompile(`\["` + regexp.QuoteMeta(key) + `"\]\s*=\s*"([^"]*)"`).FindStringSubmatch(block)
	if m == nil {
		return "", false
	}
	return m[1], true
}
func hexField(block, key string) (string, bool) {
	v, ok := fieldQuoted(block, key)
	if !ok {
		return "", false
	}
	b, e := hex.DecodeString(v)
	if e != nil {
		return "", false
	}
	return string(b), true
}
func ParseOutbox(src string) (Job, bool) {
	m := outboxBlock.FindStringSubmatch(src)
	if m == nil {
		return Job{}, false
	}
	b := m[1]
	im := regexp.MustCompile(`\["id"\]\s*=\s*(\d+)`).FindStringSubmatch(b)
	if im == nil {
		return Job{}, false
	}
	id, _ := strconv.Atoi(im[1])
	if id == 0 {
		return Job{}, false
	}
	j := Job{ID: id, Via: "reload"}
	j.Session, _ = fieldQuoted(b, "session")
	j.Chat, _ = fieldQuoted(b, "chat")
	j.Text, _ = hexField(b, "text")
	j.CWD, _ = hexField(b, "cwd")
	j.NewSession = regexp.MustCompile(`\["newSession"\]\s*=\s*true`).MatchString(b)
	if v, ok := hexField(b, "ctx"); ok {
		j.ContextText = &v
	}
	if v, ok := fieldQuoted(b, "agent"); ok {
		j.Agent = strings.ToLower(v)
	}
	if v, ok := hexField(b, "allow"); ok {
		for _, x := range strings.Split(v, FieldSep) {
			if x != "" {
				j.Allow = append(j.Allow, x)
			}
		}
	}
	for _, k := range []string{"model", "reasoning"} {
		if v, ok := fieldQuoted(b, k); ok && regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`).MatchString(v) {
			if k == "model" {
				j.Model = v
			} else {
				j.Reasoning = v
			}
		}
	}
	return j, true
}

func NormalizeGameContext(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r", ""))
	if len(s) <= ContextLimit {
		return s
	}
	limit := ContextLimit - len("\n"+ContextPartial)
	for limit > 0 && !utf8.RuneStart(s[limit]) {
		limit--
	}
	if n := strings.LastIndexByte(s[:limit], '\n'); n >= 0 {
		limit = n
	}
	return strings.TrimRight(s[:limit], " \t\n") + "\n" + ContextPartial
}

func LuaStr(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\r':
			continue
		case r == '\n':
			b.WriteString(`\n`)
		case r <= 8 || r >= 11 && r <= 31 || r == 127:
			fmt.Fprintf(&b, "\\%03d", r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

type Model struct {
	Model        string   `json:"model"`
	Name         string   `json:"name,omitempty"`
	Efforts      []string `json:"efforts,omitempty"`
	EffortsKnown *bool    `json:"effortsKnown,omitempty"`
}
type AgentSetting struct {
	Model         string   `json:"model"`
	Reasoning     string   `json:"reasoning"`
	Efforts       []string `json:"efforts"`
	CatalogStatus string   `json:"catalogStatus"`
	CatalogSource string   `json:"catalogSource"`
	EffortFlag    bool     `json:"effortFlag"`
	Models        []Model  `json:"models"`
}
type Defaults struct {
	Agent     string `json:"agent"`
	Model     string `json:"model"`
	Reasoning string `json:"reasoning"`
}
type Reply struct {
	Chat    string
	ID      int
	Status  string
	Text    string
	CWD     string
	Session string
	Agent   string
	Summary string
	Denied  []string
}
type RestoreMessage struct {
	Role  string
	ID    int
	Time  int64
	Agent string
	Text  string
}
type RestoreChat struct {
	ID       string
	Name     string
	CWD      string
	Messages []RestoreMessage
}
type Restore struct {
	Token string
	Chats []RestoreChat
}
type LuaOptions struct {
	Now           time.Time
	CWD           string
	Agent         string
	Agents        []string
	CodexModels   []Model
	AgentSettings map[string]AgentSetting
	AIDefaults    Defaults
	Restore       *Restore
}

func luaList(xs []string) string {
	a := make([]string, 0, len(xs))
	for _, s := range xs {
		a = append(a, LuaStr(s))
	}
	return strings.Join(a, ", ")
}
func LuaTable(global string, replies []Reply, opt LuaOptions) string {
	now := opt.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	var b strings.Builder
	fmt.Fprintln(&b, "-- Written by the azermind bridge. Do not edit by hand.")
	fmt.Fprintf(&b, "%s = {\n\tts = %s,\n\tnow = %d,\n\tcwd = %s,\n\tagent = %s,\n\tagents = { %s },\n", global, LuaStr(now.Format("2006-01-02T15:04:05.000Z")), now.Unix(), LuaStr(opt.CWD), LuaStr(opt.Agent), luaList(opt.Agents))
	fmt.Fprint(&b, "\tcodexModels = { ")
	for _, m := range opt.CodexModels {
		fmt.Fprintf(&b, "{ model = %s, efforts = { %s } }, ", LuaStr(m.Model), luaList(m.Efforts))
	}
	fmt.Fprintln(&b, "},")
	fmt.Fprint(&b, "\tagentSettings = { ")
	keys := make([]string, 0, len(opt.AgentSettings))
	for k := range opt.AgentSettings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, id := range keys {
		s := opt.AgentSettings[id]
		fmt.Fprintf(&b, "[%s] = { model = %s, reasoning = %s, efforts = { %s }, catalogStatus = %s, catalogSource = %s, effortFlag = %t, models = { ", LuaStr(id), LuaStr(s.Model), LuaStr(s.Reasoning), luaList(s.Efforts), LuaStr(s.CatalogStatus), LuaStr(s.CatalogSource), s.EffortFlag)
		for _, m := range s.Models {
			known := m.EffortsKnown == nil || *m.EffortsKnown
			name := m.Name
			if name == "" {
				name = m.Model
			}
			fmt.Fprintf(&b, "{ model = %s, name = %s, effortsKnown = %t, efforts = { %s } }, ", LuaStr(m.Model), LuaStr(name), known, luaList(m.Efforts))
		}
		fmt.Fprint(&b, "} }, ")
	}
	fmt.Fprintln(&b, "},")
	fmt.Fprintf(&b, "\taiDefaults = { agent = %s, model = %s, reasoning = %s },\n\treplies = {\n", LuaStr(opt.AIDefaults.Agent), LuaStr(opt.AIDefaults.Model), LuaStr(opt.AIDefaults.Reasoning))
	for _, r := range replies {
		fmt.Fprintf(&b, "\t\t{ chat = %s, id = %d, status = %s, text = %s, cwd = %s, session = %s, agent = %s,", LuaStr(r.Chat), r.ID, LuaStr(r.Status), LuaStr(r.Text), LuaStr(r.CWD), LuaStr(r.Session), LuaStr(r.Agent))
		if r.Summary != "" {
			fmt.Fprintf(&b, " summary = %s,", LuaStr(r.Summary))
		}
		if len(r.Denied) > 0 {
			fmt.Fprintf(&b, " denied = { %s },", luaList(r.Denied))
		}
		fmt.Fprintln(&b, " },")
	}
	fmt.Fprintln(&b, "\t},")
	if r := opt.Restore; r != nil {
		fmt.Fprintf(&b, "\trestore = { token = %s, chats = {\n", LuaStr(r.Token))
		for _, c := range r.Chats {
			fmt.Fprintf(&b, "\t\t{ id = %s, name = %s, cwd = %s, messages = {\n", LuaStr(c.ID), LuaStr(c.Name), LuaStr(c.CWD))
			for _, m := range c.Messages {
				fmt.Fprintf(&b, "\t\t\t{ role = %s, id = %d, t = %d, agent = %s, text = %s },\n", LuaStr(m.Role), m.ID, m.Time, LuaStr(m.Agent), LuaStr(m.Text))
			}
			fmt.Fprintln(&b, "\t\t} },")
		}
		fmt.Fprintln(&b, "\t} },")
	}
	fmt.Fprintln(&b, "}")
	return b.String()
}

func SilentWAV() []byte {
	b := make([]byte, 124)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], 116)
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], 8000)
	binary.LittleEndian.PutUint32(b[28:], 8000)
	binary.LittleEndian.PutUint16(b[32:], 1)
	binary.LittleEndian.PutUint16(b[34:], 8)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], 80)
	for i := 44; i < len(b); i++ {
		b[i] = 128
	}
	return b
}

// ResolveCWD honors the old cross-platform folder contract, including explicit Windows paths.
func ResolveCWD(raw, base string) string {
	p := strings.TrimSpace(raw)
	if p == "" {
		return base
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, _ := os.UserHomeDir()
		p = filepath.Join(home, strings.TrimLeft(p[1:], `/\`))
	}
	if regexp.MustCompile(`^[A-Za-z]:[\\/]`).MatchString(p) || strings.HasPrefix(p, `\\`) {
		return p
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(base, p))
}
