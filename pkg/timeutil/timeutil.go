package timeutil

import (
	"database/sql/driver"
	"time"
)

const (
	TimeFormat = "2006-01-02 15:04:05"
)

// JSONTime 自定义时间类型，用于JSON序列化
type JSONTime time.Time

// MarshalJSON 实现 json.Marshaler 接口
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + time.Time(t).Format(TimeFormat) + `"`), nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	str := string(data)
	if str == `""` || str == `null` {
		return nil
	}

	// 去掉引号
	if len(str) > 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	parsedTime, err := time.Parse(TimeFormat, str)
	if err != nil {
		return err
	}

	*t = JSONTime(parsedTime)
	return nil
}

// String 实现 Stringer 接口
func (t JSONTime) String() string {
	return time.Time(t).Format(TimeFormat)
}

// Value 实现 driver.Valuer 接口，用于数据库存储
func (t JSONTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (t *JSONTime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*t = JSONTime(v)
	case string:
		parsedTime, err := time.Parse(TimeFormat, v)
		if err != nil {
			return err
		}
		*t = JSONTime(parsedTime)
	case []byte:
		parsedTime, err := time.Parse(TimeFormat, string(v))
		if err != nil {
			return err
		}
		*t = JSONTime(parsedTime)
	default:
		return nil
	}

	return nil
}

// Now 返回当前时间的 JSONTime 格式
func Now() JSONTime {
	return JSONTime(time.Now())
}

// FormatTime 将 time.Time 转换为指定格式的字符串
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// ParseTime 将字符串解析为 time.Time
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse(TimeFormat, timeStr)
}
