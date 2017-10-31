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
        fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARN|FATAL] -log_dir=[string]\n", )
        flag.PrintDefaults()
        os.Exit(2)
}

func initConfig() {
	if err := config.Init(); err != nil {
		glog.Fatal(err.Error())
	}
}

func main() {
        // Check commandline options or "flags" for glog parameters
        // to be picked up by the glog module
        flag.Usage = glogUsage
        flag.Parse()

        // Since km runs as a k8 pod, log everything to stderr (stdout not supported)
        // this takes advantage of k8's logging driver allowing kubectl logs kube-monkey
	flag.Lookup("logtostderr").Value.Set("true")
	
	// Initialize configs
	initConfig()

	glog.Info("Starting kube-monkey with logging level: ", flag.Lookup("v").Value)

	if err := kubemonkey.Run(); err != nil {
		glog.Fatal(err.Error())
	}
}
