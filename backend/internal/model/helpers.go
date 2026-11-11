// MIVA-DOMAIN: learning data and pure validation rules.
package model

func Contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
