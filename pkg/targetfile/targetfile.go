package targetfile

import (
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/hirose31/ltsv-tailer/pkg/metrics"
	"github.com/hpcloud/tail"
	"github.com/lestrrat-go/strftime"
	"github.com/najeira/ltsv"
)

// TargetFile contains data to tail target file.
type TargetFile struct {
	filename        string
	currentFilename string
	strftime        *strftime.Strftime
	tailer          *tail.Tail
	metricsStore    *metrics.Store
}

// NewTargetFile creates a new TargetFile.
func NewTargetFile(filename string, metricsStore *metrics.Store) *TargetFile {
	var err error
	glog.Infof("NewTargetFile: %s", filename)
	tf := &TargetFile{filename: filename, metricsStore: metricsStore}

	if strings.Index(filename, "%") >= 0 {
		// filename contains format string
		tf.strftime, err = strftime.New(filename)
		if err != nil {
			glog.Fatal(err)
		}

		tf.currentFilename = tf.strftime.FormatString(time.Now())
		glog.Infof("currentFilename: %s (%s)", tf.currentFilename, tf.filename)
	} else {
		tf.currentFilename = filename
		glog.Infof("currentFilename: %s", tf.currentFilename)
	}

	return tf
}

func (tf *TargetFile) setCurrentTailer() {
	var err error
	tf.tailer, err = tail.TailFile(tf.currentFilename, tail.Config{
		Follow:      true,
		ReOpen:      true,
		MaxLineSize: 0,
		MustExist:   false,
		Location: &tail.SeekInfo{
			0,
			os.SEEK_END,
		},
	})
	if err != nil {
		glog.Fatal(err)
	}
}

func (tf *TargetFile) isTimestampedFile() bool {
	return tf.strftime != nil
}

func (tf *TargetFile) startTimestampChecker() {
	// interval is normally 60s
	// if contain '%M' or '%S' then 3s
	var interval time.Duration
	if strings.Index(tf.filename, "%S") >= 0 || strings.Index(tf.filename, "%M") >= 0 {
		interval = 3 * time.Second
	} else {
		interval = 60 * time.Second
	}
	glog.Infof("%s timestamp checker interval: %s", tf.currentFilename, interval)

	// checkTimestamp does not stop until exiting main process
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tf.checkTimestamp()
			}
		}
	}()
}

func (tf *TargetFile) checkTimestamp() {
	glog.V(2).Infof("checkTimestamp %s", tf.currentFilename)
	newfilename := tf.strftime.FormatString(time.Now())
	if newfilename == tf.currentFilename {
		return
	}

	glog.Infof("detect differ filename: %s %s", tf.currentFilename, newfilename)
	_, err := os.Stat(newfilename)
	if err != nil {
		glog.Infof("newfile not found. try again later: %s", newfilename)
		return
	}

	glog.Infof("found newfile: %s -> %s", tf.currentFilename, newfilename)
	tf.currentFilename = newfilename
	tf.stop()
}

// Start begins the main line processing go routine.
func (tf *TargetFile) Start() {
	if tf.isTimestampedFile() {
		tf.startTimestampChecker()
	}

	go func() {
		retryUntil := time.Unix(0, 0)
		for {
			now := time.Now()

			glog.Infof("Start: %s", tf.currentFilename)
			tf.setCurrentTailer()

			for line := range tf.tailer.Lines {
				tf.processLine(line.Text)
			}

			glog.Infof("exiting reading line loop: %s", tf.currentFilename)

			err := tf.tailer.Wait()
			if err != nil {
				glog.Warningf("failed to wait: %s", err)

				// Since logrotate creates a file with root:root 600,
				// permission is denied depending on the timing. So
				// I'm retrying for a very short time.
				if retryUntil.After(now) {
					// retryUntil is future
					glog.Warningf("rety after 0.5s. retry until %s\n", retryUntil)
					time.Sleep(time.Millisecond * 500)
				} else {
					// retryUntil is past
					if retryUntil.Before(now.Add(-5 * time.Second)) {
						// too old
						retryUntil = now.Add(time.Second * 3)
						glog.Infof("retry after 0.5s. set new retry until: %s\n", retryUntil)
						time.Sleep(time.Millisecond * 500)
					} else {
						glog.Fatalf("exceed retryUntil! so exiting...")
					}
				}
			}
		}
	}()
}

func (tf *TargetFile) stop() {
	glog.Infof("%s try to stop", tf.currentFilename)

	err := tf.tailer.Stop()
	glog.Infof("%s stopped", tf.currentFilename)
	if err != nil {
		glog.Errorf("%s Wait err: %s", tf.currentFilename, err)
	}
}

func (tf *TargetFile) processLine(line string) {
	glog.V(3).Infof("%s <%s>\n", tf.currentFilename, line)
	reader := ltsv.NewReader(strings.NewReader(line))
	records, err := reader.ReadAll()
	if err != nil {
		glog.Errorf("failed to parse LTSV: %s <%s>", err, line)
		return
	}

	for i, record := range records {
		glog.V(2).Infof("[%d]%s\n", i, record)
		tf.metricsStore.Process(record)
	}

}
