// Command harbor-wrap is the stdio MCP wrapper: agent clients talk to it as
// if it were the real server, every tool call is decided against the local
// harbor.yaml, and audit events spool locally for harbor sync to upload.
//
//	harbor-wrap --agent claude-code --target node_repl -- node_repl --flag
//
// The call path never touches the network; only harbor sync does.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/SummerXaa-Z/agent-harbor/internal/wrap"
)

func main() {
	options, command := parseFlags(os.Args[1:], os.Stderr)
	if options == nil || len(command) == 0 {
		os.Exit(2)
	}

	child := exec.Command(command[0], command[1:]...)
	child.Stderr = os.Stderr
	child.Stdin = nil
	child.Stdout = nil
	childIn, err := child.StdinPipe()
	if err != nil {
		fatal("child stdin: %v", err)
	}
	childOut, err := child.StdoutPipe()
	if err != nil {
		fatal("child stdout: %v", err)
	}
	if err := child.Start(); err != nil {
		fatal("start %s: %v", command[0], err)
	}

	// SIGINT/SIGTERM reach the whole process group from the client; forward
	// them to the child and exit with its status so restart semantics match
	// the unwrapped server.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		if child.Process != nil {
			_ = child.Process.Signal(sig)
		}
	}()

	pump := wrap.NewPump(*options)
	runErr := pump.Run(os.Stdin, os.Stdout, childIn, childOut)
	_ = childIn.Close()
	waitErr := child.Wait()

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "harbor-wrap: %v\n", runErr)
	}
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "harbor-wrap: child: %v\n", waitErr)
		os.Exit(1)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "harbor-wrap: "+format+"\n", args...)
	os.Exit(1)
}
