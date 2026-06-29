package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/kamichidu/go-netclip/internal/daemon"
)

const defaultAddr = "127.0.0.1:45555"

//go:embed usage.txt
var usageString string

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "daemon":
		fs := flag.NewFlagSet("daemon", flag.ExitOnError)
		listenAddr := fs.String("listen", defaultAddr, "TCP address to listen on")
		background := fs.Bool("background", false, "Run the daemon process in the background")
		clipCmd := fs.String("clipboard-command", "", "Explicit command to use for copying (e.g. wl-copy)")
		pasteCmd := fs.String("paste-command", "", "Explicit command to use for pasting (e.g. wl-paste)")

		fs.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: netclip daemon [options]\n\nOptions:\n")
			fs.PrintDefaults()
		}

		if err := fs.Parse(os.Args[2:]); err != nil {
			log.Fatalf("error parsing flags: %v", err)
		}

		runDaemon(*listenAddr, *background, *clipCmd, *pasteCmd)

	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, usageString)
}

func runDaemon(addr string, background bool, clipboardCommand string, pasteCommand string) {
	// 1. Check if already listening (guarantees idempotency)
	if isAlreadyRunning(addr) {
		log.Printf("netclip daemon is already running and listening on %s. Exiting successfully.", addr)
		os.Exit(0)
	}

	// 2. Handle background startup
	if background {
		startBackgroundDaemon(addr, clipboardCommand, pasteCommand)
		os.Exit(0)
	}

	// 3. Normal startup (start HTTP server)
	srv := daemon.NewServer(addr, clipboardCommand, pasteCommand)
	if err := srv.Start(); err != nil {
		log.Fatalf("failed to start daemon: %v", err)
	}
}

func isAlreadyRunning(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

func startBackgroundDaemon(addr, clipboardCommand, pasteCommand string) {
	self, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get current executable path: %v", err)
	}

	var args []string
	args = append(args, "daemon", "--listen", addr)
	if clipboardCommand != "" {
		args = append(args, "--clipboard-command", clipboardCommand)
	}
	if pasteCommand != "" {
		args = append(args, "--paste-command", pasteCommand)
	}

	cmd := exec.Command(self, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start background process: %v", err)
	}

	// Verify background startup by polling with timeout
	success := false
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		if isAlreadyRunning(addr) {
			success = true
			break
		}
	}

	if !success {
		log.Fatalf("failed to confirm daemon startup within 500ms. Check logs or bind permissions.")
	}

	log.Printf("netclip daemon successfully started in background on %s", addr)
}
