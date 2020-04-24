package vm

import (
	"strings"
	"testing"
)

func TestTimeSeries_Write(t *testing.T) {
	var testCases = []struct {
		name string
		ts   *TimeSeries
		exp  string
	}{
		{
			name: "one datapoint",
			ts: &TimeSeries{
				Name: "foo",
				LabelPairs: []LabelPair{
					{
						Name:  "key",
						Value: "val",
					},
				},
				Timestamps: []int64{1577877162200},
				Values:     []float64{1},
			},
			exp: `{"metric":{"__name__":"foo","key":"val"},"timestamps":[1577877162200],"values":[1]}`,
		},
		{
			name: "multiple datapoints",
			ts: &TimeSeries{
				Name: "foo",
				LabelPairs: []LabelPair{
					{
						Name:  "key",
						Value: "val",
					},
				},
				Timestamps: []int64{1577877162200, 15778771622400, 15778771622600},
				Values:     []float64{1, 1.6263, 32.123},
			},
			exp: `{"metric":{"__name__":"foo","key":"val"},"timestamps":[1577877162200,15778771622400,15778771622600],"values":[1,1.6263,32.123]}`,
		},
		{
			name: "escape",
			ts: &TimeSeries{
				Name: "foo \\",
				LabelPairs: []LabelPair{
					{
						Name:  "escaped \\\\\\ key",
						Value: "val \\",
					},
				},
				Timestamps: []int64{1577877162200},
				Values:     []float64{1},
			},
			exp: `{"metric":{"__name__":"foo \\","escaped \\\\\\ key":"val \\"},"timestamps":[1577877162200],"values":[1]}`,
		},
		{
			name: "no datapoints",
			ts: &TimeSeries{
				Name: "foo",
				LabelPairs: []LabelPair{
					{
						Name:  "key",
						Value: "val",
					},
				},
			},
			exp: ``,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b := make([]byte, 0)
			b = tc.ts.write(b)
			got := strings.TrimSpace(string(b))
			if got != tc.exp {
				t.Fatalf("\ngot:  %q\nwant: %q", got, tc.exp)
			}
		})
	}
}
