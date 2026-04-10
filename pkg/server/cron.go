package server

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/xbapps/xbvr/pkg/api"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/session"
	"github.com/xbapps/xbvr/pkg/tasks"
)

var cronInstance *cron.Cron
var rescrapTask cron.EntryID
var rescanTask cron.EntryID
var previewTask cron.EntryID
var actorScrapeTask cron.EntryID
var stashdbScrapeTask cron.EntryID
var linkScenesTask cron.EntryID
var pmvMatchTask cron.EntryID

func cronTaskLog(task string, fields logrus.Fields) *logrus.Entry {
	base := logrus.Fields{
		"task":    task,
		"trigger": "cron",
	}
	for key, value := range fields {
		base[key] = value
	}
	return log.WithFields(base)
}

func logCronSchedule(task string, schedule string) {
	cronTaskLog(task, logrus.Fields{"schedule": schedule}).Info("scheduled")
}

func logNextCronRun(task string, entryID cron.EntryID) {
	if entryID == 0 {
		cronTaskLog(task, nil).Debug("no cron entry configured")
		return
	}
	nextRun := cronInstance.Entry(entryID).Next
	if nextRun.IsZero() {
		cronTaskLog(task, nil).Info("next_run unavailable")
		return
	}
	cronTaskLog(task, logrus.Fields{"next_run": nextRun.Format(time.RFC3339)}).Info("next_run")
}

func runCronTask(task string, entryID cron.EntryID, fn func()) {
	tlog := cronTaskLog(task, nil)
	if session.HasActiveSession() {
		tlog.Info("skipped active_session=true")
		logNextCronRun(task, entryID)
		return
	}

	started := time.Now()
	tlog.Info("started")
	fn()
	tlog.WithField("duration", time.Since(started).Round(time.Millisecond).String()).Info("finished")
	logNextCronRun(task, entryID)
}

func SetupCron() {
	cronInstance = cron.New()
	cronInstance.AddFunc("@every 2s", session.CheckForDeadSession)
	cronInstance.AddFunc("@every 6h", tasks.CalculateCacheSizes)
	if config.Config.Cron.RescrapeSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.RescrapeSchedule))
		logCronSchedule("rescrape", schedule)
		rescrapTask, _ = cronInstance.AddFunc(schedule, scrapeCron)
	}
	if config.Config.Cron.RescanSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.RescanSchedule))
		logCronSchedule("rescan", schedule)
		rescanTask, _ = cronInstance.AddFunc(schedule, rescanCron)
	}
	if config.Config.Cron.PreviewSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.PreviewSchedule))
		logCronSchedule("preview-generate", schedule)
		previewTask, _ = cronInstance.AddFunc(schedule, generatePreviewCron)
	}
	if config.Config.Cron.ActorRescrapeSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.ActorRescrapeSchedule))
		logCronSchedule("actor-rescrape", schedule)
		actorScrapeTask, _ = cronInstance.AddFunc(schedule, actorRescrapeCron)
	}
	if config.Config.Cron.StashdbRescrapeSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.StashdbRescrapeSchedule))
		logCronSchedule("stashdb-rescrape", schedule)
		stashdbScrapeTask, _ = cronInstance.AddFunc(schedule, stashdbRescrapeCron)
	}
	if config.Config.Cron.LinkScenesSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.LinkScenesSchedule))
		logCronSchedule("link-scenes", schedule)
		linkScenesTask, _ = cronInstance.AddFunc(schedule, linkScenesCron)
	}
	if config.Config.Cron.PmvMatchSchedule.Enabled {
		schedule := formatCronSchedule(config.CronSchedule(config.Config.Cron.PmvMatchSchedule))
		logCronSchedule("pmv-match-unmatched", schedule)
		pmvMatchTask, _ = cronInstance.AddFunc(schedule, pmvMatchCron)
	}
	cronInstance.Start()

	go tasks.CalculateCacheSizes()

	if config.Config.Cron.RescrapeSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.RescrapeSchedule.RunAtStartDelay)*time.Minute, scrapeCron)
	}
	if config.Config.Cron.RescanSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.RescanSchedule.RunAtStartDelay)*time.Minute, rescanCron)
	}
	if config.Config.Cron.PreviewSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.PreviewSchedule.RunAtStartDelay)*time.Minute, generatePreviewCron)
	}
	if config.Config.Cron.ActorRescrapeSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.ActorRescrapeSchedule.RunAtStartDelay)*time.Minute, actorRescrapeCron)
	}
	if config.Config.Cron.StashdbRescrapeSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.StashdbRescrapeSchedule.RunAtStartDelay)*time.Minute, stashdbRescrapeCron)
	}
	if config.Config.Cron.LinkScenesSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.LinkScenesSchedule.RunAtStartDelay)*time.Minute, linkScenesCron)
	}
	if config.Config.Cron.PmvMatchSchedule.RunAtStartDelay > 0 {
		time.AfterFunc(time.Duration(config.Config.Cron.PmvMatchSchedule.RunAtStartDelay)*time.Minute, pmvMatchCron)
	}

	logNextCronRun("rescrape", rescrapTask)
	logNextCronRun("rescan", rescanTask)
	logNextCronRun("preview-generate", previewTask)
	logNextCronRun("actor-rescrape", actorScrapeTask)
	logNextCronRun("stashdb-rescrape", stashdbScrapeTask)
	logNextCronRun("link-scenes", linkScenesTask)
	logNextCronRun("pmv-match-unmatched", pmvMatchTask)
}

func scrapeCron() {
	runCronTask("rescrape", rescrapTask, func() {
		tasks.Scrape("_enabled", "", "")
	})
}

func rescanCron() {
	runCronTask("rescan", rescanTask, func() {
		tasks.RescanVolumes(-1)
	})
}
func actorRescrapeCron() {
	runCronTask("actor-rescrape", actorScrapeTask, func() {
		tasks.ScrapeActors()
	})
}
func stashdbRescrapeCron() {
	runCronTask("stashdb-rescrape", stashdbScrapeTask, func() {
		api.StashdbRunAll()
	})
}

func linkScenesCron() {
	runCronTask("link-scenes", linkScenesTask, func() {
		tasks.MatchAlternateSources()
	})
}

func pmvMatchCron() {
	if session.HasActiveSession() {
		cronTaskLog("pmv-match-unmatched", nil).Info("skipped active_session=true")
		logNextCronRun("pmv-match-unmatched", pmvMatchTask)
		return
	}

	common.StartAsyncTask("pmv-match-unmatched", "cron", logrus.Fields{
		"limit": 200,
	}, func() {
		tasks.RunPMVMatchUnmatchedTask(tasks.PMVMatchBatchRequest{
			DryRun: false,
			Limit:  200,
		})
	})
	logNextCronRun("pmv-match-unmatched", pmvMatchTask)
}

var previewGenerateInProgress = false

func generatePreviewCron() {
	if session.HasActiveSession() {
		cronTaskLog("preview-generate", nil).Info("skipped active_session=true")
		logNextCronRun("preview-generate", previewTask)
		return
	}
	if previewGenerateInProgress {
		cronTaskLog("preview-generate", nil).Info("skipped already_running=true")
		logNextCronRun("preview-generate", previewTask)
		return
	}

	previewGenerateInProgress = true
	defer func() {
		previewGenerateInProgress = false
	}()

	started := time.Now()
	tlog := cronTaskLog("preview-generate", nil)
	tlog.Info("started")

	if !config.Config.Cron.PreviewSchedule.UseRange {
		tasks.GeneratePreviews(nil)
	} else {
		endTime := calcEndTime(config.Config.Cron.PreviewSchedule.HourStart, config.Config.Cron.PreviewSchedule.HourEnd, config.Config.Cron.PreviewSchedule.MinuteStart)
		tlog.WithField("end_time", endTime.Format(time.RFC3339)).Info("using stop window")
		tasks.GeneratePreviews(&endTime)
	}

	tlog.WithField("duration", time.Since(started).Round(time.Millisecond).String()).Info("finished")
	logNextCronRun("preview-generate", previewTask)
}
func formatCronSchedule(schedule config.CronSchedule) string {
	// 	this routine will format a crontab range description, https://crontab.guru is a good tool to decode the range description generated
	// 	if the start hour > end hour then the time range will extend across midnight into the next day
	//		to achieve this with cron you create a range from the start until midnight and then a second from from midnight to the end time
	//		we need to calculate the start time for the range after midnight to make sure we still get the right iterval
	hourInterval := ""
	formattedHourSchedule := ""

	if !schedule.UseRange {
		return fmt.Sprintf("@every %vh", schedule.HourInterval)
	}

	if schedule.HourInterval > 0 {
		hourInterval = fmt.Sprintf("/%v", schedule.HourInterval)
	}
	if schedule.HourStart > schedule.HourEnd { // if the start > end, time range goes over midnight into the next day
		afterMidnightStart := (schedule.HourInterval - ((24 - schedule.HourStart) % schedule.HourInterval)) % schedule.HourInterval // calculate what time after midnight to restart
		if afterMidnightStart <= schedule.HourEnd {
			// schedule the range needed to start after midnight
			formattedHourSchedule = fmt.Sprintf("%v-%v%v,%v-23%v", afterMidnightStart, schedule.HourEnd, hourInterval, schedule.HourStart, hourInterval)
		} else {
			// the interval was too big to schedule after midnight before reaching the end time, so only create the pre midnight range
			formattedHourSchedule = fmt.Sprintf("%v-23%v", schedule.HourStart, hourInterval)
		}
	} else {
		formattedHourSchedule = fmt.Sprintf("%v-%v%v", schedule.HourStart, schedule.HourEnd, hourInterval)
	}
	return fmt.Sprintf("%v %v * * *", schedule.MinuteStart, formattedHourSchedule)
}
func calcEndTime(startHour int, endHour int, minuteStart int) time.Time {

	dt := time.Now()
	if startHour > endHour {
		if dt.Hour() > endHour {
			return time.Date(dt.Year(), dt.Month(), dt.Day(), 23, 59, 0, 0, dt.Location())
		} else {
			return time.Date(dt.Year(), dt.Month(), dt.Day(), endHour, minuteStart, 0, 0, dt.Location())
		}
	} else {
		return time.Date(dt.Year(), dt.Month(), dt.Day(), endHour, minuteStart, 0, 0, dt.Location())
	}
}
