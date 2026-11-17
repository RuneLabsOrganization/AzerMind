package state

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/transport"
)

type Context struct {
	Session string `json:"session"`
	Text    string `json:"text"`
}
type Store struct {
	sync.Mutex      `json:"-"`
	File            string                    `json:"-"`
	LastID          int                       `json:"lastId"`
	Sessions        map[string]string         `json:"sessions"`
	Handled         map[string]map[string]int `json:"handled"`
	Seen            map[string]int64          `json:"seen"`
	SessionCWD      map[string]string         `json:"sessionCwd"`
	SessionAgent    map[string]string         `json:"sessionAgent"`
	SessionSettings map[string]string         `json:"sessionSettings"`
	Presence        int                       `json:"presence"`
	Context         *Context                  `json:"context,omitempty"`
}

func Open(path string) (*Store, error) {
	s := &Store{File: path}
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(b) > 0 {
		// Node's older state used handled[session] as a numeric high-water mark.
		// Convert that format before decoding the typed Go representation.
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(b, &fields); err != nil {
			return nil, err
		}
		handled := fields["handled"]
		delete(fields, "handled")
		base, err := json.Marshal(fields)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(base, s); err != nil {
			return nil, err
		}
		s.Handled = make(map[string]map[string]int)
		var raw map[string]json.RawMessage
		if len(handled) > 0 {
			if err = json.Unmarshal(handled, &raw); err != nil {
				return nil, err
			}
		}
		for token, v := range raw {
			if len(v) > 0 && v[0] == '{' {
				var m map[string]int
				if err = json.Unmarshal(v, &m); err != nil {
					return nil, err
				}
				s.Handled[token] = m
				continue
			}
			var n int
			if err = json.Unmarshal(v, &n); err != nil {
				return nil, err
			}
			m := map[string]int{}
			for i := 1; i <= n && i <= 1000; i++ {
				m[strconv.Itoa(i)] = 1
			}
			s.Handled[token] = m
		}
	}
	if s.Sessions == nil {
		s.Sessions = map[string]string{}
	}
	if s.Handled == nil {
		s.Handled = map[string]map[string]int{}
	}
	if s.Seen == nil {
		s.Seen = map[string]int64{}
	}
	if s.SessionCWD == nil {
		s.SessionCWD = map[string]string{}
	}
	if s.SessionAgent == nil {
		s.SessionAgent = map[string]string{}
	}
	if s.SessionSettings == nil {
		s.SessionSettings = map[string]string{}
	}
	return s, nil
}
func (s *Store) Save() error { s.Lock(); defer s.Unlock(); return s.saveLocked() }
func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return transport.AtomicWrite(s.File, append(b, '\n'))
}
func (s *Store) SeenJob(j protocol.Job) bool {
	s.Lock()
	defer s.Unlock()
	if h := s.Handled[j.Session]; h != nil {
		return h[strconv.Itoa(j.ID)] != 0
	}
	return j.Session == "" && j.ID <= s.LastID
}
func (s *Store) Mark(j protocol.Job) error {
	s.Lock()
	defer s.Unlock()
	h := s.Handled[j.Session]
	if h == nil {
		h = map[string]int{}
		s.Handled[j.Session] = h
	}
	h[strconv.Itoa(j.ID)] = 1
	if len(h) > 1000 {
		ids := make([]int, 0, len(h))
		for k := range h {
			n, _ := strconv.Atoi(k)
			ids = append(ids, n)
		}
		sort.Ints(ids)
		for i := 0; i < len(ids)-1000; i++ {
			delete(h, strconv.Itoa(ids[i]))
		}
	}
	if j.ID > s.LastID {
		s.LastID = j.ID
	}
	s.Seen[j.Session] = time.Now().UnixMilli()
	return s.saveLocked()
}
func (s *Store) Forget(j protocol.Job) error {
	s.Lock()
	defer s.Unlock()
	delete(s.Sessions, protocol.SessionKey(j))
	delete(s.Sessions, protocol.ChatKey(j))
	delete(s.SessionCWD, protocol.SessionKey(j))
	delete(s.SessionAgent, protocol.SessionKey(j))
	delete(s.SessionSettings, protocol.SessionKey(j))
	return s.saveLocked()
}
func (s *Store) SetContext(j protocol.Job) error {
	if j.ContextText == nil {
		return nil
	}
	s.Lock()
	defer s.Unlock()
	text := protocol.NormalizeGameContext(*j.ContextText)
	if text == "" {
		s.Context = nil
	} else {
		s.Context = &Context{Session: j.Session, Text: text}
	}
	return s.saveLocked()
}
func (s *Store) Prune(now time.Time) error {
	s.Lock()
	defer s.Unlock()
	cutoff := now.Add(-30 * 24 * time.Hour).UnixMilli()
	for token := range s.Handled {
		if token == "" {
			continue
		}
		if ts := s.Seen[token]; ts == 0 {
			s.Seen[token] = now.UnixMilli()
		} else if ts < cutoff {
			delete(s.Handled, token)
			delete(s.Seen, token)
		}
	}
	return s.saveLocked()
}
func (s *Store) GetSession(j protocol.Job, agent, cwd, settings string) (string, error) {
	s.Lock()
	defer s.Unlock()
	key := protocol.SessionKey(j)
	old := s.SessionSettings[key]
	if j.NewSession || (old != "" && old != settings) || (s.SessionAgent[key] != "" && s.SessionAgent[key] != agent) || (s.SessionCWD[key] != "" && s.SessionCWD[key] != cwd) {
		delete(s.Sessions, key)
		delete(s.Sessions, protocol.ChatKey(j))
	}
	s.SessionSettings[key] = settings
	resume := s.Sessions[key]
	if resume == "" {
		resume = s.Sessions[protocol.ChatKey(j)]
	}
	return resume, s.saveLocked()
}
func (s *Store) SaveSession(j protocol.Job, id, agent, cwd string) error {
	if id == "" {
		return nil
	}
	s.Lock()
	defer s.Unlock()
	key := protocol.SessionKey(j)
	s.Sessions[key] = id
	s.SessionAgent[key] = agent
	s.SessionCWD[key] = cwd
	return s.saveLocked()
}

// ContextText returns a snapshot safe to read while the capture loop updates it.
func (s *Store) ContextText() string {
	s.Lock()
	defer s.Unlock()
	if s.Context == nil {
		return ""
	}
	return s.Context.Text
}
func (s *Store) SetPresence(n int) error {
	s.Lock()
	defer s.Unlock()
	s.Presence = n
	return s.saveLocked()
}
