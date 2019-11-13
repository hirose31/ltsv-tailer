package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/golang/glog"
	"github.com/hirose31/ltsv-tailer/pkg/metrics"
	"github.com/hirose31/ltsv-tailer/pkg/targetfile"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "net/http/pprof"
)

type buildInfo struct {
	Branch   string
	Version  string
	Revision string
}

func (b buildInfo) String() string {
	return fmt.Sprintf(
		"ltsv-tailer version %s git revision %s go version %s go arch %s go os %s",
		b.Version,
		b.Revision,
		runtime.Version(),
		runtime.GOARCH,
		runtime.GOOS,
	)
}

// StringSet is the set of string
type StringSet []string

func (ss *StringSet) String() string {
	return fmt.Sprintf("%q", *ss)
}

// Set string to StringSet
func (ss *StringSet) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

var (
	showVersion       = flag.Bool("version", false, "Print version information.")
	enablePprof       = flag.Bool("pprof", false, "Enable net/http/pprof (port 6060)")
	files             StringSet
	metricsConfigFile = flag.String("metrics", "", "Metrics config file")
)

// supplied by the linker
var (
	Branch   string
	Version  string
	Revision string
)

func init() {
	flag.Var(&files, "file", "File to tail. This option may be spcified multiple times")
}

func main() {
	buildInfo := buildInfo{
		Branch:   Branch,
		Version:  Version,
		Revision: Revision,
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage:
  %s [OPTIONS] ARGS...
Options:
`,
			os.Args[0],
		)
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println(buildInfo.String())
		os.Exit(1)
	}
	glog.Info(buildInfo.String())
	glog.Infof("Commandline: %q", os.Args)

	if files == nil {
		flag.Usage()
		glog.Exitf("missing -file option")
	}
	if *metricsConfigFile == "" {
		flag.Usage()
		glog.Exitf("missing -metrics option")
	}

	enableDumpProfile()

	if *enablePprof {
		glog.Infof("Start pprof")
		go func() {
			glog.Fatal(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	metricsStore := metrics.NewStore()
	metricsStore.Load(*metricsConfigFile)

	for _, filename := range files {
		tf := targetfile.NewTargetFile(filename, metricsStore)
		tf.Start()
	}

	glog.Infof("Start promhttp")
	http.Handle("/metrics", promhttp.Handler())
	glog.Fatal(http.ListenAndServe(":9588", nil))

}

func enableDumpProfile() {
	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, syscall.SIGQUIT)

		for {
			select {
			case <-sig:
				fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------\n")
				fmt.Fprintf(os.Stderr, "* goroutine\n")
				pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "* heap\n")
				pprof.Lookup("heap").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "* allocs\n")
				pprof.Lookup("allocs").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------\n")
			}
		}
	}()
}
