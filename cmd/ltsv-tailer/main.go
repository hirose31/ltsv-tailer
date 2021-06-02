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

	_ "net/http/pprof" // #nosec G108
)

const version = "0.1.2"

var revision = "HEAD"

type buildInfo struct {
	Version  string
	Revision string
}

func (b buildInfo) String() string {
	return fmt.Sprintf(
		"ltsv-tailer %s (rev: %s/%s)",
		b.Version,
		b.Revision,
		runtime.Version(),
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

func init() {
	flag.Var(&files, "file", "File to tail. This option may be spcified multiple times")
}

func main() {
	buildInfo := buildInfo{
		Version:  version,
		Revision: revision,
	}

	listenAddr := ":9588"

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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>LTSV tailer</title></head>
			<body>
			<h1>LTSV tailer</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
		// fixme handle error
	})
	glog.Infof("Listening on %s", listenAddr)
	glog.Fatal(http.ListenAndServe(listenAddr, nil))

}

func enableDumpProfile() {
	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, syscall.SIGQUIT)

		for {
			select {
			case <-sig:
				fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------\n")
				fmt.Fprintf(os.Stderr, "* goroutine\n") // #nosec G104
				pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "* heap\n") // #nosec G104
				pprof.Lookup("heap").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "* allocs\n") // #nosec G104
				pprof.Lookup("allocs").WriteTo(os.Stderr, 1)
				fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------\n") // #nosec G104
			}
		}
	}()
}
