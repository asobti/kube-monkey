package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubemonkey"
)

func glogUsage() {
	fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func initLogging() {
	// Check commandline options or "flags" for glog parameters
	// to be picked up by the glog module
	flag.Usage = glogUsage
	flag.Parse()

	// Since km runs as a k8 pod, log everything to stderr (stdout not supported)
	// this takes advantage of k8's logging driver allowing kubectl logs kube-monkey
	if err := flag.Lookup("alsologtostderr").Value.Set("true"); err != nil {
		glog.Errorf("Failed to set alsologtostderr. Error: %v", err)
	}

	// glog only opens log files when -logtostderr is off, and it writes them to
	// the system temp dir when -log_dir is empty. Neither case needs a directory
	// prepared here.
	if flag.Lookup("logtostderr").Value.String() == "true" {
		return
	}
	logDir := flag.Lookup("log_dir").Value.String()
	if logDir == "" {
		return
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
			// The image runs on an empty filesystem owned by root, so a non-root
			// user cannot create this directory. Mount a writable volume at the
			// path, or drop -log_dir and log to stderr only.
			glog.Errorf("Failed to create log directory at %s; glog will fall back to %s. Error: %v", logDir, os.TempDir(), err)
		} else {
			glog.V(5).Infof("Created custom logging %s directory!", logDir)
		}
	}
}

func initConfig() {
	if err := config.Init(); err != nil {
		glog.Fatal(err.Error())
	}

	// glog stamps every line with the process local time, so without this the
	// line prefix and the times inside the messages would sit in different
	// zones. Set once here, before any goroutine starts, because time.Local is
	// read without locking everywhere else. A later config reload does not move
	// it; that needs a restart.
	time.Local = config.Timezone()
}

func main() {
	// Initialize logging
	initLogging()

	// Initialize configs
	initConfig()

	if logDir := flag.Lookup("log_dir").Value.String(); logDir != "" {
		glog.V(1).Infof("Starting kube-monkey with v logging level %v and local log directory %s", flag.Lookup("v").Value, logDir)
	} else {
		glog.V(1).Infof("Starting kube-monkey with v logging level %v", flag.Lookup("v").Value)
	}

	if err := kubemonkey.Run(); err != nil {
		glog.Fatal(err.Error())
	}
}
