package common

//DifferentElementStrings :
// find different elements between 2 strings
func DifferentElementStrings(src1, src2 []string) []string {
	res := []string{}
	m := map[string]bool{}
	lSrc := []string{}
	sSrc := []string{}
	if len(src1) >= len(src2) {
		lSrc = src1
		sSrc = src2
	} else {
		lSrc = src2
		sSrc = src1
	}
	for _, v := range sSrc {
		m[v] = true
	}
	for _, v := range lSrc {
		if !m[v] {
			res = append(res, v)
		}
	}
	return res
}
