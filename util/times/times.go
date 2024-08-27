package times

import "time"

type Duration struct {
	t1 int64
	t2 int64
}

func DurationFromTime(t1 int64, t2 int64) Duration {
	return Duration{
		t1: t1,
		t2: t2,
	}
}

func DurationFromNow(t1 int64) Duration {
	return Duration{
		t1: t1,
		t2: time.Now().Unix(),
	}
}

// 是否跨天
func (d Duration) IsCrossDay() bool {
	y1, m1, d1 := time.Unix(d.t1, 0).Date()
	y2, m2, d2 := time.Unix(d.t2, 0).Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return false
	} else {
		return true
	}
}

// 是否跨周
func (d Duration) IsCrossWeek() bool {
	y1, w1 := time.Unix(d.t1, 0).ISOWeek()
	y2, w2 := time.Unix(d.t2, 0).ISOWeek()
	if y1 == y2 && w1 == w2 {
		return false
	} else {
		return true
	}
}

// 是否跨月
func (d Duration) IsCrossMonth() bool {
	y1, m1, _ := time.Unix(d.t1, 0).Date()
	y2, m2, _ := time.Unix(d.t2, 0).Date()
	if y1 == y2 && m1 == m2 {
		return false
	} else {
		return true
	}
}

// 间隔的天数，以0点开始计算
func (d Duration) PassedDays() int64 {
	t1 := time.Unix(d.t1, 0)
	t2 := time.Unix(d.t2, 0)
	d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, t1.Location())
	d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, t2.Location())
	return (d2.Unix() - d1.Unix()) / 86400
}

// 以固定天数作为周期，计算当前的周期数
// 周期从1开始计算
func (d Duration) GetCycle(days int64) int64 {
	passedDays := d.PassedDays()
	return (passedDays % days) + 1
}

func TodayBeginTime() int64 {
	t := time.Now()
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.Unix()
}

func TodayEndTime() int64 {
	t := time.Now()
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.Unix() + 86400 - 1
}
