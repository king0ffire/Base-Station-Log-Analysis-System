package accounting

import "sort"

type Eventinfo struct {
	Eventname string
	Count     int
}

type Categoryinfo struct {
	Categoryname string
	Count        int
	Eventinfo    []*Eventinfo
}

func NewCategoryinfo(categoryname string) *Categoryinfo {
	return &Categoryinfo{
		Categoryname: categoryname,
		Count:        0,
		Eventinfo:    []*Eventinfo{},
	}
}

func (c *Categoryinfo) Getcategorycount() int {
	return c.Count
}

func (e *Eventinfo) DerefEventinfo() Eventinfo {
	return *e
}

func (c *Categoryinfo) AddEvent(eventname string, eventcount int) {
	c.Eventinfo = append(c.Eventinfo, &Eventinfo{
		Eventname: eventname,
		Count:     eventcount,
	})
}

func (c *Categoryinfo) SortEvent() {
	sort.Slice(c.Eventinfo, func(i, j int) bool {
		return c.Eventinfo[i].Count > c.Eventinfo[j].Count
	})
}
