// Package profiling owns the lifecycle of the Go CPU and heap profiles.
package profiling

import (
	"errors"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
)

// Start begins a CPU profile into cpuFile. The returned stop ends it and,
// when memFile isn't empty, writes a heap profile there (after a garbage
// collection, so it shows live memory), then logs the process's memory
// totals: all bytes allocated, bytes obtained from the OS (close to the
// peak heap), and the number of garbage collections.
func Start(cpuFile, memFile string) (func() error, error) {
	file, err := os.Create(cpuFile)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return func() error {
		pprof.StopCPUProfile()
		err := file.Close()
		if memFile != "" {
			err = errors.Join(err, writeHeap(memFile))
		}
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		slog.Info("memory", "total_alloc_mb", stats.TotalAlloc>>20, "sys_mb", stats.Sys>>20, "num_gc", stats.NumGC)
		return err
	}, nil
}

func writeHeap(name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	runtime.GC()
	return errors.Join(pprof.WriteHeapProfile(file), file.Close())
}
