package desktopapp

import (
	"sync"
	"time"

	"github.com/ltaoo/velo/notification"
	"github.com/rs/zerolog"
)

// ReminderScheduler periodically scans open tasks for due reminders and fires
// system notifications. It keeps an in-memory set of already-fired reminder
// keys to avoid duplicate notifications within a session, and also persists
// the fired flag back to disk so reminders don't re-fire across restarts.
type ReminderScheduler struct {
	logger *zerolog.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup
	fired  map[string]bool
	mu     sync.Mutex
}

func NewReminderScheduler(logger *zerolog.Logger) *ReminderScheduler {
	return &ReminderScheduler{
		logger: logger,
		stopCh: make(chan struct{}),
		fired:  make(map[string]bool),
	}
}

func (s *ReminderScheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Run once immediately on start.
		s.tick()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.tick()
			}
		}
	}()
	s.logger.Info().Msg("reminder scheduler started")
}

func (s *ReminderScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	s.logger.Info().Msg("reminder scheduler stopped")
}

func (s *ReminderScheduler) tick() {
	ctx, err := requireActiveVault()
	if err != nil {
		return
	}

	tasks, err := listVaultTasks(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("reminder: failed to list tasks")
		return
	}

	now := time.Now()

	for i := range tasks {
		task := &tasks[i]
		if task.Status != taskStatusOpen {
			continue
		}
		if len(task.Reminders) == 0 {
			continue
		}

		dirty := false
		for j := range task.Reminders {
			reminder := &task.Reminders[j]
			if reminder.Fired {
				continue
			}

			fireAt := s.resolveFireTime(task, reminder)
			if fireAt.IsZero() {
				continue
			}

			key := task.ID + "|" + reminderKey(reminder)
			s.mu.Lock()
			alreadyFired := s.fired[key]
			s.mu.Unlock()
			if alreadyFired {
				continue
			}

			if !fireAt.After(now) {
				// Fire the notification.
				title := "提醒: " + task.Title
				body := reminderBody(task, reminder)
				if err := notification.Show(notification.Options{
					Title: title,
					Body:  body,
					Sound: true,
				}); err != nil {
					s.logger.Warn().Err(err).Str("task", task.ID).Msg("reminder: notification failed")
				} else {
					s.logger.Info().Str("task", task.ID).Str("title", task.Title).Msg("reminder fired")
				}

				s.mu.Lock()
				s.fired[key] = true
				s.mu.Unlock()

				reminder.Fired = true
				dirty = true
			}
		}

		if dirty {
			if err := writeTaskRecord(ctx, *task); err != nil {
				s.logger.Warn().Err(err).Str("task", task.ID).Msg("reminder: failed to persist fired state")
			}
		}
	}
}

func (s *ReminderScheduler) resolveFireTime(task *TaskRecord, reminder *TaskReminder) time.Time {
	switch reminder.Type {
	case "absolute":
		return parseMemoTime(reminder.At)
	case "relative":
		baseTime := s.resolveBaseTime(task, reminder.Base)
		if baseTime.IsZero() {
			return time.Time{}
		}
		return baseTime.Add(-time.Duration(reminder.OffsetMinutes) * time.Minute)
	default:
		return time.Time{}
	}
}

func (s *ReminderScheduler) resolveBaseTime(task *TaskRecord, base string) time.Time {
	switch base {
	case "dueAt":
		return parseMemoTime(task.DueAt)
	case "startAt":
		return parseMemoTime(task.StartAt)
	default:
		// Default to dueAt if base is empty.
		if task.DueAt != "" {
			return parseMemoTime(task.DueAt)
		}
		return parseMemoTime(task.StartAt)
	}
}

func reminderKey(r *TaskReminder) string {
	if r.Type == "absolute" {
		return "abs:" + r.At
	}
	return "rel:" + r.Base + ":" + intToStr(r.OffsetMinutes)
}

func reminderBody(task *TaskRecord, r *TaskReminder) string {
	if r.Type == "relative" && r.OffsetMinutes > 0 {
		return task.Title + " 将在 " + formatMinutes(r.OffsetMinutes) + " 后到期"
	}
	return task.Title
}

func formatMinutes(minutes int) string {
	if minutes >= 1440 && minutes%1440 == 0 {
		days := minutes / 1440
		if days == 1 {
			return "1 天"
		}
		return intToStr(days) + " 天"
	}
	if minutes >= 60 && minutes%60 == 0 {
		hours := minutes / 60
		if hours == 1 {
			return "1 小时"
		}
		return intToStr(hours) + " 小时"
	}
	return intToStr(minutes) + " 分钟"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
