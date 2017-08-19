// PromHouse
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package storages

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
)

// Storage represents generic storage.
type Storage interface {
	// Read runs queries in the storage and returns the same amount of matrixes.
	// Event if they are empty, they must be present in the returned slice.
	Read(context.Context, []Query) (*prompb.ReadResponse, error)

	// Write puts data into storage.
	Write(context.Context, *prompb.WriteRequest) error

	prometheus.Collector
}

// Query represents query against stored data.
type Query struct {
	Start    model.Time
	End      model.Time
	Matchers []Matcher
}

func (q Query) String() string {
	res := fmt.Sprintf("[%d,%d,{", q.Start, q.End)
	matchers := make([]string, len(q.Matchers))
	for i, m := range q.Matchers {
		matchers[i] = m.String()
	}
	res += strings.Join(matchers, ",") + "}]"
	return res
}

type MatchType int

const (
	MatchEqual MatchType = iota
	MatchNotEqual
	MatchRegexp
	MatchNotRegexp
)

func (m MatchType) String() string {
	switch m {
	case MatchEqual:
		return "="
	case MatchNotEqual:
		return "!="
	case MatchRegexp:
		return "=~"
	case MatchNotRegexp:
		return "!~"
	default:
		panic("unknown match type")
	}
}

type Matcher struct {
	Name  model.LabelName
	Type  MatchType
	Value string
	re    *regexp.Regexp
}

func (m Matcher) String() string {
	return fmt.Sprintf("%s%s%q", m.Name, m.Type, m.Value)
}

func (m *Matcher) Match(metric model.Metric) bool {
	if (m.re == nil) && (m.Type == MatchRegexp || m.Type == MatchNotRegexp) {
		m.re = regexp.MustCompile("^(?:" + m.Value + ")$")
	}

	switch m.Type {
	case MatchEqual:
		return string(metric[m.Name]) == m.Value
	case MatchNotEqual:
		return string(metric[m.Name]) != m.Value
	case MatchRegexp:
		return m.re.MatchString(string(metric[m.Name]))
	case MatchNotRegexp:
		return !m.re.MatchString(string(metric[m.Name]))
	default:
		panic("unknown match type")
	}
}

// sortTimeSeries sorts timeseries by a value of __name__ label.
func sortTimeSeries(timeSeries []*prompb.TimeSeries) {
	sort.Slice(timeSeries, func(i, j int) bool {
		var nameI, nameJ string
		for _, l := range timeSeries[i].Labels {
			if l.Name == model.MetricNameLabel {
				nameI = l.Value
				break
			}
		}
		for _, l := range timeSeries[j].Labels {
			if l.Name == model.MetricNameLabel {
				nameJ = l.Value
				break
			}
		}
		return nameI < nameJ
	})
}

// sortLabels sorts labels by name.
func sortLabels(labels []*prompb.Label) {
	sort.Slice(labels, func(i, j int) bool { return labels[i].Name < labels[j].Name })
}

// check interfaces
var (
	_ fmt.Stringer = Query{}
	_ fmt.Stringer = Matcher{}
	_ fmt.Stringer = MatchType(0)
)
