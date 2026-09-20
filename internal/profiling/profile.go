// Package profiling owns the lifecycle of a Go CPU profile.
package profiling

import (
	"errors"
	"os"
	"runtime/pprof"
)

func Start(filename string) (func() error, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return func() error { pprof.StopCPUProfile(); return file.Close() }, nil
}
