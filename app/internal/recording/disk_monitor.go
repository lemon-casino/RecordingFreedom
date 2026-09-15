package recording

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lemon-casino/RecordingFreedom/app/internal/appdata"
	"github.com/lemon-casino/RecordingFreedom/app/internal/recpackage"
)

const (
	// EnvMinFreeDiskMB overrides the free-space threshold that auto-stops an
	// active recording. Values <= 0 disable the monitor.
	EnvMinFreeDiskMB = "RECORDINGFREEDOM_MIN_FREE_DISK_MB"

	diskMonitorInterval = 5 * time.Second
)

func minFreeDiskBytes() uint64 {
	value := strings.TrimSpace(os.Getenv(EnvMinFreeDiskMB))
	if value == "" {
		return appdata.MinimumRecommendedVideoFreeBytes
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0
	}
	return uint64(parsed) * 1024 * 1024
}

type diskMonitor struct {
	stop chan struct{}
	done chan struct{}
}

func (m *diskMonitor) close() {
	if m == nil {
		return
	}
	select {
	case <-m.stop:
	default:
		close(m.stop)
	}
	<-m.done
}

type diskMonitorOptions struct {
	packageDir   string
	manifestPath string
	threshold    uint64
	interval     time.Duration
	freeBytes    func(path string) (uint64, error)
	packages     *recpackage.Service
	onStop       func()
}

func runDiskMonitor(m *diskMonitor, opts diskMonitorOptions) {
	defer close(m.done)
	ticker := time.NewTicker(opts.interval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
		}
		free, err := opts.freeBytes(opts.packageDir)
		if err != nil || free >= opts.threshold {
			continue
		}
		annotateLowDiskStop(opts.packages, opts.manifestPath, free, opts.threshold)
		// onStop runs asynchronously: a synchronous call into Service.Stop
		// would deadlock against stopDiskMonitor waiting on m.done.
		stop := opts.onStop
		if stop != nil {
			go stop()
		}
		return
	}
}

func annotateLowDiskStop(packages *recpackage.Service, manifestPath string, free uint64, threshold uint64) {
	if packages == nil {
		return
	}
	manifest, err := packages.ReadManifest(manifestPath)
	if err != nil {
		return
	}
	manifest.Diagnostics.Message = fmt.Sprintf(
		"Recording stopped automatically: free disk space %.0f MB is below the %.0f MB threshold.",
		float64(free)/(1024*1024), float64(threshold)/(1024*1024))
	_ = packages.WriteManifest(manifestPath, manifest)
}
