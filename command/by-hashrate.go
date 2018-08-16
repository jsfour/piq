package command

type ByHashrate []SummaryRes

func (a ByHashrate) Len() int           { return len(a) }
func (a ByHashrate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHashrate) Less(i, j int) bool { return a[i].Summary[0].GhsAv < a[j].Summary[0].GhsAv }
