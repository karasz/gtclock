package glibtai

import "time"

type leapsecond struct {
	begin  time.Time
	offset int
}

var leapseconds = []*leapsecond{
	{time.Date(1972, time.July, 1, 0, 0, 0, 0, time.UTC), 11},
	{time.Date(1973, time.January, 1, 0, 0, 0, 0, time.UTC), 12},
	{time.Date(1974, time.January, 1, 0, 0, 0, 0, time.UTC), 13},
	{time.Date(1975, time.January, 1, 0, 0, 0, 0, time.UTC), 14},
	{time.Date(1976, time.January, 1, 0, 0, 0, 0, time.UTC), 15},
	{time.Date(1977, time.January, 1, 0, 0, 0, 0, time.UTC), 16},
	{time.Date(1978, time.January, 1, 0, 0, 0, 0, time.UTC), 17},
	{time.Date(1979, time.January, 1, 0, 0, 0, 0, time.UTC), 18},
	{time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC), 19},
	{time.Date(1981, time.July, 1, 0, 0, 0, 0, time.UTC), 20},
	{time.Date(1982, time.July, 1, 0, 0, 0, 0, time.UTC), 21},
	{time.Date(1983, time.July, 1, 0, 0, 0, 0, time.UTC), 22},
	{time.Date(1985, time.July, 1, 0, 0, 0, 0, time.UTC), 23},
	{time.Date(1988, time.January, 1, 0, 0, 0, 0, time.UTC), 24},
	{time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC), 25},
	{time.Date(1991, time.January, 1, 0, 0, 0, 0, time.UTC), 26},
	{time.Date(1992, time.July, 1, 0, 0, 0, 0, time.UTC), 27},
	{time.Date(1993, time.July, 1, 0, 0, 0, 0, time.UTC), 28},
	{time.Date(1994, time.July, 1, 0, 0, 0, 0, time.UTC), 29},
	{time.Date(1996, time.January, 1, 0, 0, 0, 0, time.UTC), 30},
	{time.Date(1997, time.July, 1, 0, 0, 0, 0, time.UTC), 31},
	{time.Date(1999, time.January, 1, 0, 0, 0, 0, time.UTC), 32},
	{time.Date(2006, time.January, 1, 0, 0, 0, 0, time.UTC), 33},
	{time.Date(2009, time.January, 1, 0, 0, 0, 0, time.UTC), 34},
	{time.Date(2012, time.July, 1, 0, 0, 0, 0, time.UTC), 35},
	{time.Date(2015, time.July, 1, 0, 0, 0, 0, time.UTC), 36},
	{time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC), 37},
}

func lsoffset(t time.Time) uint64 {
	for i := len(leapseconds) - 1; i >= 0; i-- {
		ls := leapseconds[i]
		if t.Unix() >= ls.begin.Unix() {
			return uint64(ls.offset)
		}
	}

	return 0
}
