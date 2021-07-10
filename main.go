package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/e11jah/air-force/ping"
	"github.com/e11jah/air-force/subscribe"
	"github.com/spf13/cobra"
)

var (
	debug       bool
	showVersion bool
	version     string
	gitCommit   string

	url         string
	counter     int
	pingTimeout string
	interval    string
	sigs        chan os.Signal
)

func init() {
	rootCmd.Flags().BoolVarP(&debug, "debug", "D", false, "show the debug log")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show the version and exit")
	rootCmd.Flags().IntVarP(&counter, "counter", "c", 4, "ping counter")
	rootCmd.Flags().StringVarP(&pingTimeout, "timeout", "T", "1s", `connect timeout, units are "ns", "us" (or "µs"), "ms", "s", "m", "h"`)
	rootCmd.Flags().StringVarP(&interval, "interval", "I", "500ms", `ping interval, units are "ns", "us" (or "µs"), "ms", "s", "m", "h"`)
}

var rootCmd = cobra.Command{
	Use:   "test tcp ping speed for nodes of subscribe url",
	Short: "",
	Long:  "",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("version: %s\n", version)
			fmt.Printf("git: %s\n", gitCommit)
			return
		}
		if len(args) != 1 {
			cmd.Usage()
			return
		}
		var (
			timeoutDuration  time.Duration
			intervalDuration time.Duration
		)

		if res, err := strconv.Atoi(pingTimeout); err == nil {
			timeoutDuration = time.Duration(res) * time.Millisecond
		} else {
			timeoutDuration, err = time.ParseDuration(pingTimeout)
			if err != nil {
				fmt.Printf("parse timeout failed, %s\n", err)
				cmd.Usage()
				return
			}
		}

		if res, err := strconv.Atoi(interval); err == nil {
			intervalDuration = time.Duration(res) * time.Millisecond
		} else {
			intervalDuration, err = time.ParseDuration(interval)
			if err != nil {
				fmt.Printf("parse interval failed, %s\n", err)
				cmd.Usage()
				return
			}
		}

		if !debug {
			log.SetOutput(ioutil.Discard)
		}

		url = args[0]
		ctx := context.Background()
		configs, err := subscribe.LoadConfigs(ctx, url)
		if err != nil {
			fmt.Printf("can not subscribe url '%s', err: %s\n", url, err)
			return
		}
		if len(configs) < 1 {
			fmt.Printf("can not load configs from url '%s', plz see debug info", url)
			return
		}

		totalTimeout := timeoutDuration * time.Duration(counter*len(configs))
		ctx, cancel := context.WithTimeout(ctx, totalTimeout)
		defer cancel()

		pinger := ping.NewTCPing(1)
		defer pinger.Close()

		for _, cfg := range configs {
			port, err := strconv.Atoi(cfg.Port)
			if err != nil {
				log.Printf("parse port of config '%v' error, %s\n", cfg, err)
				continue
			}

			go pinger.AsyncPing(ctx, &ping.Target{
				Protocol: ping.TCP,
				Name:     cfg.Name,
				Host:     cfg.Addr,
				Port:     port,
				Counter:  counter,
				Interval: intervalDuration,
				Timeout:  timeoutDuration,
			})
		}

		resultCh := pinger.GetResult()
		cnt := len(configs)
		var fastestResult *ping.Result

	loop:
		for {
			select {
			case <-sigs:
				fmt.Println("exit")
				return
			case <-ctx.Done():
				fmt.Printf("timeout: %s, context exit: %s\n", totalTimeout, ctx.Err())
				return
			case result := <-resultCh:
				cnt--
				if fastestResult == nil {
					fastestResult = result
				} else if result.Success >= fastestResult.Success && result.Avg() < fastestResult.Avg() {
					fastestResult = result
				}
				if cnt <= 0 {
					break loop
				}
			}
		}
		fmt.Printf("\nfastest ping result: %s\n", fastestResult)
	},
}

func main() {
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
