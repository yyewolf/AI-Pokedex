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
