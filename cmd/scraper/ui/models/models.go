package models

import "time"

type StatsData struct {
	Targets      int
	PendingQueue int
	TotalPages   int
	RecentErrors int
}

type TargetData struct {
	ID         int64
	WebsiteURL string
	SitemapURL string
	Status     string
	CreatedAt  time.Time
}

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
	URL       string
	Details   string
}

func (l LogEntry) LevelColor() string {
	switch l.Level {
	case "error":
		return "red"
	case "warn":
		return "yellow"
	case "info":
		return "blue"
	case "success":
		return "green"
	default:
		return "gray"
	}
}

func (l LogEntry) LevelIcon() string {
	switch l.Level {
	case "error":
		return "exclamation-circle"
	case "warn":
		return "exclamation-triangle"
	case "info":
		return "info-circle"
	case "success":
		return "check-circle"
	default:
		return "circle"
	}
}

type HealthData struct {
	Status    string
	Service   string
	Timestamp time.Time
	Uptime    string
}
