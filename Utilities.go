package main

type list []string

func (l list) Has(item string) bool {
	for _, checking := range l {
		if item == checking {
			return true
		}
	}
	return false
}

func (l list) Add(item string) {
	l = append(l, item)
}

func (l list) Remove(item string) {
	for i, v := range l {
		if v == item {
			l[i] = l[len(l)-1]
			l = l[:len(l)-1]
			break
		}
	}
}
