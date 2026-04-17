package common

import (
	"context"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/gammazero/nexus/v3/client"
	"github.com/shiena/ansicolor"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var Log = *logrus.New()
var taskRunSeq uint64

type WampHook struct {
	publisher *client.Client
}

func NewWampHook() *WampHook {
	wh := &WampHook{}

	publisher, _ := client.ConnectNet(context.Background(), "ws://"+WsAddr+"/ws", client.Config{
		Realm: "default",
	})

	wh.publisher = publisher

	return wh
}

func (hook *WampHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *WampHook) Fire(entry *logrus.Entry) error {
	err := hook.publisher.Publish("service.log", nil, nil, map[string]interface{}{
		"level":     entry.Level.String(),
		"message":   entry.Message,
		"data":      entry.Data,
		"timestamp": entry.Time.Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}
	return nil
}

func InitLogging() {
	//	Log.Out = os.Stdout
	Log.SetLevel(logrus.InfoLevel)
	if EnvConfig.Debug {
		Log.SetLevel(logrus.DebugLevel)
	}

	Log.Formatter = &prefixed.TextFormatter{
		ForceColors:     true,
		ForceFormatting: true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	}

	//	create / open log file in AppDir folder
	lfile, err := os.OpenFile(AppDir+"/xbvr.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		//		defer lfile.Close()
		mw := io.MultiWriter(lfile, os.Stdout)
		Log.Out = ansicolor.NewAnsiColorWriter(mw)
	} else {
		Log.Info("Failed to log to file, using default stderr")
	}
}

func WithTaskLog(task string, fields logrus.Fields) *logrus.Entry {
	taskFields := logrus.Fields{
		"task": task,
	}
	for key, value := range fields {
		taskFields[key] = value
	}
	return Log.WithFields(taskFields)
}

func StartAsyncTask(task string, trigger string, fields logrus.Fields, fn func()) uint64 {
	runID := atomic.AddUint64(&taskRunSeq, 1)
	taskFields := logrus.Fields{
		"trigger": trigger,
		"run_id":  runID,
	}
	for key, value := range fields {
		taskFields[key] = value
	}

	entry := WithTaskLog(task, taskFields)
	entry.Info("queued")

	go func() {
		started := time.Now()
		entry.Info("started")
		defer func() {
			duration := time.Since(started).Round(time.Millisecond)
			if recovered := recover(); recovered != nil {
				entry.WithFields(logrus.Fields{
					"duration": duration.String(),
					"panic":    recovered,
				}).Error("panicked")
				panic(recovered)
			}
			entry.WithField("duration", duration.String()).Info("finished")
		}()

		fn()
	}()

	return runID
}
