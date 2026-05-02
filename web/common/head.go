package common

type NavAsideEntry struct {
	Name string
	Link string
}

type NavAsideEntryList []NavAsideEntry

func (ls NavAsideEntryList) HasNextBlank(i int) bool {
	if i >= len(ls)-1 {
		return false
	}
	return ls[i+1].Name == ""
}

func (ls NavAsideEntryList) HasPrevBlank(i int) bool {
	if i <= 0 {
		return false
	}
	return ls[i-1].Name == ""
}
