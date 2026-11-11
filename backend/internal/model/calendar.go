// MIVA-DOMAIN: learning data and pure validation rules.
package model

import (
	"strings"
	"time"
)

var jakarta = time.FixedZone("WIB", 7*60*60)

func DayKey(t time.Time) string { return t.In(jakarta).Format("2006-01-02") }

func Monday(t time.Time) string {
	d := t.In(jakarta)
	return DayKey(d.AddDate(0, 0, -(int(d.Weekday())+6)%7))
}

func Calendar(s Settings, id, origin string, now time.Time) string {
	date := DayKey(now)
	start, _ := time.Parse(time.RFC3339, date+"T"+s.Reminder+":00+07:00")
	for i := 0; i < 8; i++ {
		selected := false
		for _, d := range s.Days {
			if d == (int(start.In(jakarta).Weekday())+6)%7 {
				selected = true
			}
		}
		if selected && start.After(now) {
			break
		}
		start = start.AddDate(0, 0, 1)
	}
	duration := (s.WeeklyMinutes + len(s.Days) - 1) / len(s.Days)
	format := func(t time.Time) string { return t.UTC().Format("20060102T150405Z") }
	codes := []string{}
	for _, i := range s.Days {
		codes = append(codes, []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}[i])
	}
	escape := strings.NewReplacer("\\", "\\\\", "\n", "\\n", ",", "\\,", ";", "\\;")
	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Miva//ID",
		"CALSCALE:GREGORIAN",
		"BEGIN:VEVENT",
		"UID:english-" + id + "@english-garden.local",
		"DTSTAMP:" + format(now),
		"DTSTART:" + format(start),
		"DTEND:" + format(start.Add(time.Duration(duration)*time.Minute)),
		"RRULE:FREQ=WEEKLY;BYDAY=" + strings.Join(codes, ","),
		"SUMMARY:Belajar Miva",
		"DESCRIPTION:" + escape.Replace(s.Goal+"\n"+origin),
		"BEGIN:VALARM",
		"TRIGGER:PT0M",
		"ACTION:DISPLAY",
		"DESCRIPTION:Waktunya belajar bahasa Inggris",
		"END:VALARM",
		"END:VEVENT",
		"END:VCALENDAR",
	}
	for i, line := range lines {
		var out strings.Builder
		n := 0
		for _, r := range line {
			b := string(r)
			if n+len(b) > 73 {
				out.WriteString("\r\n ")
				n = 1
			}
			out.WriteString(b)
			n += len(b)
		}
		lines[i] = out.String()
	}
	return strings.Join(lines, "\r\n") + "\r\n"
}
