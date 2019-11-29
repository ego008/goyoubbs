package util

import (
	"strconv"
	"time"
)

func TimeFmt(tp interface{}, sample string, tz int) string {
	offset := int64(time.Duration(tz) * time.Hour)
	var t int64
	switch tp.(type) {
	case uint64:
		t = int64(tp.(uint64))
	case string:
		i64, err := strconv.ParseInt(tp.(string), 10, 64)
		if err != nil {
			return ""
		}
		t = i64
	case int64:
		t = tp.(int64)
	}
	if len(sample) == 0 {
		sample = "2006-01-02 15:04:05"
	}
	tm := time.Unix(t, offset).UTC()
	return tm.Format(sample)
}

func TimeHuman(ts interface{}) string {
	var t int64
	switch ts.(type) {
	case uint64:
		t = int64(ts.(uint64))
	case string:
		i64, err := strconv.ParseInt(ts.(string), 10, 64)
		if err != nil {
			return ""
		}
		t = i64
	case int64:
		t = ts.(int64)
	}

	then := time.Unix(t, 0)
	diff := time.Now().UTC().Sub(then)

	hours := diff.Hours()
	days := int(hours) / 24
	if days > 0 {
		switch {
		case days >= 365:
			y := days / 365
			d := days % 365
			if d == 0 {
				return strconv.Itoa(y) + "年前"
			}
			return strconv.Itoa(y) + "年" + strconv.Itoa(d) + "天前"
		case days >= 30:
			m := days / 30
			d := days % 30
			if d == 0 {
				return strconv.Itoa(m) + "月前"
			}
			return strconv.Itoa(m) + "月" + strconv.Itoa(d) + "天前"
		case days >= 7:
			w := days / 7
			d := days % 7
			if d == 0 {
				return strconv.Itoa(w) + "周前"
			}
			return strconv.Itoa(w) + "周" + strconv.Itoa(d) + "天前"
		case days >= 1:
			h := int(hours) % 24
			if h == 0 {
				return strconv.Itoa(days) + "天前"
			}
			return strconv.Itoa(days) + "天" + strconv.Itoa(h) + "小时前"
		default:
			return "1天前"
		}
	}

	seconds := diff.Seconds()
	switch {
	case seconds >= 3600:
		h := int(seconds / 3600)
		m := int(seconds) % 3600
		if m == 0 {
			return strconv.Itoa(h) + "小时前"
		}
		return strconv.Itoa(h) + "小时" + strconv.Itoa(m/60) + "分前"
	case seconds >= 60:
		m := int(seconds / 60)
		s := int(seconds) % 60
		if s == 0 {
			return strconv.Itoa(m) + "分钟前"
		}
		return strconv.Itoa(m) + "分" + strconv.Itoa(s) + "秒前"
	}

	return "刚刚"
}
