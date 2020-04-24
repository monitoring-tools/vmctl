package vm

import (
	"fmt"
	"io"
	"log"
	"strconv"
)

type TimeSeries struct {
	Name       string
	LabelPairs []LabelPair
	Timestamps []int64
	Values     []interface{}
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

// cWriter used to avoid error checking
// while doing Write calls.
// cWriter caches the first error if any
// and discards all sequential write calls
type cWriter struct {
	w   io.Writer
	n   int
	err error
}

func (cw *cWriter) append(p []byte) {
	if cw.err != nil {
		return
	}
	n, err := cw.w.Write(p)
	cw.n += n
	cw.err = err
}

//"{"metric":{"__name__":"cpu_usage_guest","arch":"x64","hostname":"host_19",},"timestamps":[1567296000000,1567296010000],"values":[1567296000000,66]}
func (ts *TimeSeries) write(w io.Writer) (int, error) {
	pointsCount := len(ts.Timestamps)
	if pointsCount == 0 {
		return 0, nil
	}

	buf := make([]byte, 0)
	cw := &cWriter{w: w}

	cw.append([]byte(`{"metric":{"__name__":`))
	buf = FastEscape(buf, ts.Name)
	if len(ts.LabelPairs) > 0 {
		for _, lp := range ts.LabelPairs {
			buf = append(buf, ',')
			buf = FastEscape(buf, lp.Name)
			buf = append(buf, ':')
			buf = FastEscape(buf, lp.Value)
		}
	}
	cw.append(buf)

	buf = buf[:0]
	cw.append([]byte(`},"timestamps":[`))
	for i := 0; i < pointsCount; i++ {
		if i != 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendInt(buf, ts.Timestamps[i], 10)
	}
	cw.append(buf)

	buf = buf[:0]
	cw.append([]byte(`],"values":[`))
	for i := 0; i < pointsCount; i++ {
		if i != 0 {
			buf = append(buf, ',')
		}

		val := ts.Values[i]

		valFloat, ok := val.(float64)
		if ok {
			buf = strconv.AppendFloat(buf, valFloat, 'f', -1, 64)
			continue
		}

		valInt, ok := val.(int)
		if ok {
			buf = strconv.AppendInt(buf, int64(valInt), 10)
			continue
		}

		log.Panicf("unknown type for value: %v", val)
	}
	cw.append(buf)
	cw.append([]byte("]}\n"))

	return cw.n, cw.err
}
