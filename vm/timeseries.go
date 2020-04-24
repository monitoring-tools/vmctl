package vm

import (
	"fmt"
	"strconv"
)

type TimeSeries struct {
	Name       string
	LabelPairs []LabelPair
	Timestamps []int64
	Values     []float64
}

type LabelPair struct {
	Name  string
	Value string
}

func (ts TimeSeries) String() string {
	s := ts.Name
	if len(ts.LabelPairs) < 1 {
		return s
	}
	var labels string
	for i, lp := range ts.LabelPairs {
		labels += fmt.Sprintf("%s=%q", lp.Name, lp.Value)
		if i < len(ts.LabelPairs)-1 {
			labels += ","
		}
	}
	return fmt.Sprintf("%s{%s}", s, labels)
}

//"{"metric":{"__name__":"cpu_usage_guest","arch":"x64","hostname":"host_19",},"timestamps":[1567296000000,1567296010000],"values":[1567296000000,66]}
func (ts *TimeSeries) write(buf []byte) []byte {
	pointsCount := len(ts.Timestamps)
	if pointsCount == 0 {
		return buf
	}

	buf = append(buf, []byte(`{"metric":{"__name__":`)...)
	buf = FastEscape(buf, ts.Name)
	if len(ts.LabelPairs) > 0 {
		for _, lp := range ts.LabelPairs {
			buf = append(buf, ',')
			buf = FastEscape(buf, lp.Name)
			buf = append(buf, ':')
			buf = FastEscape(buf, lp.Value)
		}
	}

	buf = append(buf, []byte(`},"timestamps":[`)...)
	for i := 0; i < pointsCount; i++ {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendInt(buf, ts.Timestamps[i], 10)
	}

	buf = append(buf, []byte(`],"values":[`)...)
	for i := 0; i < pointsCount; i++ {
		if i != 0 {
			buf = append(buf, ',')
		}

		buf = strconv.AppendFloat(buf, ts.Values[i], 'f', -1, 64)
	}
	buf = append(buf, []byte("]}\n")...)

	return buf
}
