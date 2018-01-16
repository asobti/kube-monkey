package main

import (
	"fmt"
	"os"
	"flag"
	
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubemonkey"
)

func glogUsage() {
	fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARN|FATAL] -logs_dir=[string]\n", )
	flag.PrintDefaults()
	os.Exit(2)
}

func initLogging() {
	// Check commandline options or "flags" for glog parameters
	// to be picked up by the glog module
	flag.Usage = glogUsage
	flag.Parse()

	if _, err := os.Stat(flag.Lookup("logs_dir").Value.String()); os.IsNotExist(err) {
		err = os.MkdirAll(flag.Lookup("logs_dir").Value.String(), os.ModePerm)
		if (err != nil) {
			glog.Errorf("Failed to open custom log directory; defaulting to /tmp! Error: %v", flag.Lookup("logs_dir").Value, err)
		} else {
			glog.V(5).Infof("Created custom logging %s directory!", flag.Lookup("logs_dir").Value)
		}
	}
	// Since km runs as a k8 pod, log everything to stderr (stdout not supported)
	// this takes advantage of k8's logging driver allowing kubectl logs kube-monkey
	flag.Lookup("alsologtostderr").Value.Set("true")
}

func initConfig() {
	if err := config.Init(); err != nil {
		glog.Fatal(err.Error())
	}
}

func main() {
	// Initialize logging
	initLogging()
	
	// Initialize configs
	initConfig()
	
	glog.V(1).Infof("Starting kube-monkey with v logging level %v and local log directory %s", flag.Lookup("v").Value, flag.Lookup("logs_dir").Value)
	
	if err := kubemonkey.Run(); err != nil {
		glog.Fatal(err.Error())
	}
}
