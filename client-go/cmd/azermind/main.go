package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/bridge"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/config"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/protocol"
	"github.com/mateusmaaia/AzerMind-wow/client-go/internal/transport"
)

func main() { os.Exit(run()) }
func run() int {
	var conf, project, inject, agent string
	var once, install bool
	flag.StringVar(&conf, "config", "", "Path to bridge/config.json")
	flag.StringVar(&project, "project", "", "Default project folder")
	flag.StringVar(&inject, "inject", "", "Inject a prompt instead of reading the WoW addon")
	flag.StringVar(&agent, "agent", "", "Agent for --inject")
	flag.BoolVar(&once, "once", false, "Process one pending SavedVariables job and exit")
	flag.BoolVar(&install, "install-slots", false, "Create missing addon transport slots")
	flag.Parse()
	if conf == "" {
		if v := os.Getenv("AZERMIND_CONFIG"); v != "" {
			conf = v
		} else {
			conf = filepath.Join("bridge", "config.json")
		}
	}
	conf, err := filepath.Abs(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	cfg, err := config.Load(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if install {
		made, kept, err := transport.InstallSlots(cfg.Transport())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("slots: %d files created, %d kept\n", made, kept)
		return 0
	}
	if project == "" {
		project = os.Getenv("AZERMIND_PROJECT")
	}
	if project == "" {
		project = cfg.DefaultCWD
	}
	if project == "" {
		project, _ = os.Getwd()
	}
	project, err = filepath.Abs(project)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	cl, err := bridge.New(cfg, filepath.Dir(conf), project)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if inject != "" {
		if agent == "" {
			agent = cfg.Agent
		}
		cl.Submit(protocol.Job{ID: cl.State.LastID + 1, Chat: "", Text: inject, Via: "inject", Agent: strings.ToLower(agent)})
		cl.Wait()
		return 0
	}
	if once {
		cl.PollSavedVariables()
		cl.Wait()
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Printf("AzerMind Go bridge | project: %s | addon: %s | agent: %s\n", project, cfg.AddonDir, cfg.Agent)
	cl.Run(ctx)
	return 0
}
