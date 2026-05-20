// Package toolbox provides small in-process diagnostic helpers built on the
// runtime/pprof and runtime/debug packages. None of these helpers call
// os.Exit; failures are returned as errors so that callers can decide how
// to surface them.
package toolbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"
)

var startTime = time.Now()

func pid() int { return os.Getpid() }

// ProcessInput parses a simple command-line directive and routes it to the
// matching helper. Unknown commands are reported as an error.
//
// Recognised commands:
//
//	lookup goroutine|heap|threadcreate|block
//	get cpuprof
//	get memprof
//	gc summary
func ProcessInput(ctx context.Context, input string, w io.Writer) error {
	switch input {
	case "lookup goroutine":
		return pprof.Lookup("goroutine").WriteTo(w, 2)
	case "lookup heap":
		return pprof.Lookup("heap").WriteTo(w, 2)
	case "lookup threadcreate":
		return pprof.Lookup("threadcreate").WriteTo(w, 2)
	case "lookup block":
		return pprof.Lookup("block").WriteTo(w, 2)
	case "get cpuprof":
		return GetCPUProfile(ctx, w, 30*time.Second)
	case "get memprof":
		return MemProf(w)
	case "gc summary":
		_, err := fmt.Fprintln(w, PrintGCSummary())
		return err
	default:
		return fmt.Errorf("toolbox: unknown command %q", input)
	}
}

// MemProf writes a heap profile to ./mem-<pid>.memprof and reports the path
// (plus the pprof command to inspect it) on w.
//
// The file is created mode 0600 because heap profiles routinely include
// secrets, JWTs, and other sensitive in-memory state.
func MemProf(w io.Writer) error {
	filename := "mem-" + strconv.Itoa(pid()) + ".memprof"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("toolbox: create %s: %w", filename, err)
	}
	defer func() { _ = f.Close() }()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("toolbox: write heap profile: %w", err)
	}
	fmt.Fprintf(w, "created heap profile %s\n", filename)
	_, exe := path.Split(os.Args[0])
	fmt.Fprintf(w, "inspect with: go tool pprof %s %s\n", exe, filename)
	return nil
}

// GetCPUProfile records a CPU profile of duration to ./cpu-<pid>.pprof.
// Cancellation via ctx stops the profile early.
//
// The file is created mode 0600; CPU profiles can leak the values of
// in-flight HTTP requests, including authorization headers.
func GetCPUProfile(ctx context.Context, w io.Writer, duration time.Duration) error {
	filename := "cpu-" + strconv.Itoa(pid()) + ".pprof"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("toolbox: create %s: %w", filename, err)
	}
	defer func() { _ = f.Close() }()

	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf("toolbox: start cpu profile: %w", err)
	}
	defer pprof.StopCPUProfile()

	t := time.NewTimer(duration)
	defer t.Stop()
	select {
	case <-ctx.Done():
		if err := ctx.Err(); !errors.Is(err, context.Canceled) {
			return fmt.Errorf("toolbox: cpu profile interrupted: %w", err)
		}
	case <-t.C:
	}

	fmt.Fprintf(w, "created cpu profile %s\n", filename)
	_, exe := path.Split(os.Args[0])
	fmt.Fprintf(w, "inspect with: go tool pprof %s %s\n", exe, filename)
	return nil
}

// PrintGCSummary returns a one-line snapshot of GC and memory statistics.
func PrintGCSummary() string {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	gcstats := &debug.GCStats{PauseQuantiles: make([]time.Duration, 100)}
	debug.ReadGCStats(gcstats)
	return formatGC(memStats, gcstats)
}

func formatGC(memStats *runtime.MemStats, gcstats *debug.GCStats) string {
	elapsed := time.Since(startTime)
	allocRate := float64(memStats.TotalAlloc) / elapsed.Seconds()

	if gcstats.NumGC == 0 {
		return fmt.Sprintf("Alloc:%s Sys:%s Alloc(Rate):%s/s",
			humanBytes(memStats.Alloc),
			humanBytes(memStats.Sys),
			humanBytes(uint64(allocRate)),
		)
	}

	lastPause := gcstats.Pause[0]
	overhead := float64(gcstats.PauseTotal) / float64(elapsed) * 100
	return fmt.Sprintf(
		"NumGC:%d Pause:%s Pause(Avg):%s Overhead:%.2f%% Alloc:%s Sys:%s Alloc(Rate):%s/s Histogram:%s %s %s",
		gcstats.NumGC,
		humanDuration(lastPause),
		humanDuration(avgDuration(gcstats.Pause)),
		overhead,
		humanBytes(memStats.Alloc),
		humanBytes(memStats.Sys),
		humanBytes(uint64(allocRate)),
		humanDuration(gcstats.PauseQuantiles[94]),
		humanDuration(gcstats.PauseQuantiles[98]),
		humanDuration(gcstats.PauseQuantiles[99]),
	)
}

func avgDuration(items []time.Duration) time.Duration {
	if len(items) == 0 {
		return 0
	}
	var sum time.Duration
	for _, item := range items {
		sum += item
	}
	return sum / time.Duration(len(items))
}

func humanBytes(b uint64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.2fK", float64(b)/1024)
	case b < 1024*1024*1024:
		return fmt.Sprintf("%.2fM", float64(b)/(1024*1024))
	default:
		return fmt.Sprintf("%.2fG", float64(b)/(1024*1024*1024))
	}
}

func humanDuration(d time.Duration) string {
	if d == 0 {
		return "0"
	}
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fus", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.2fm", d.Minutes())
	default:
		return fmt.Sprintf("%.2fh", d.Hours())
	}
}
