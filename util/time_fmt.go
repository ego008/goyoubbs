package util

import (
	"strconv"
	"time"
)

// TimeFmt 格式化时间
func TimeFmt(t int64, s string) string {
	if s == "" {
		s = "2006-01-02 15:04:05"
	}
	// time.Unix(t, 0).UTC().Add(offsetTime).Format(s)
	return time.Unix(t, 0).UTC().Format(s) // 存取时间戳时已经加上 offsetTime 这里就不再加
}

// GetCNTM 取8时区时间戳
func GetCNTM(timeOffSet time.Duration) int64 {
	// 10位数
	return time.Now().UTC().Add(timeOffSet).Unix()
}

func TimeHuman(ts interface{}, timeOffSet time.Duration) string {
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
	diff := time.Now().UTC().Add(timeOffSet).Sub(then)

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

// U+1F566 -- &#x1f566;
var clockUnicode = map[string]string{
	"1:0":  "&#x1F550;",
	"2:0":  "&#x1F551;",
	"3:0":  "&#x1F552;",
	"4:0":  "&#x1F553;",
	"5:0":  "&#x1F554;",
	"6:0":  "&#x1F555;",
	"7:0":  "&#x1F556;",
	"8:0":  "&#x1F557;",
	"9:0":  "&#x1F558;",
	"10:0": "&#x1F559;",
	"11:0": "&#x1F55A;",
	"12:0": "&#x1F55B;",
	"1:1":  "&#x1F55C;",
	"2:1":  "&#x1F55D;",
	"3:1":  "&#x1F55E;",
	"4:1":  "&#x1F55F;",
	"5:1":  "&#x1F560;",
	"6:1":  "&#x1F561;",
	"7:1":  "&#x1F562;",
	"8:1":  "&#x1F563;",
	"9:1":  "&#x1F564;",
	"10:1": "&#x1F565;",
	"11:1": "&#x1F566;",
	"12:1": "&#x1F567;",
}

func GetTimeUnicodeClock(tp interface{}) string {
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

	//tt := time.Unix(t, int64(offsetTime)).UTC()
	tt := time.Unix(t, 0).UTC()
	hourStr := strconv.Itoa(tt.Hour()%12 + 1)   //[1, 12]
	minuteStr := strconv.Itoa(tt.Minute() / 30) // 0|1

	return clockUnicode[hourStr+":"+minuteStr]
}
