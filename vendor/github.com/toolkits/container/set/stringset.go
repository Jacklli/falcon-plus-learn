package set

type StringSet struct {
	M map[string]struct{}
}
/*
创建一个set（map[string]struct{}），存储string
 */
func NewStringSet() *StringSet {
	return &StringSet{
		M: make(map[string]struct{}),
	}
}
/*
讲一个string加入set，这里是一个"host:port"
 */
func (this *StringSet) Add(elt string) *StringSet {
	this.M[elt] = struct{}{}
	return this
}

func (this *StringSet) Exists(elt string) bool {
	_, exists := this.M[elt]
	return exists
}

func (this *StringSet) Delete(elt string) {
	delete(this.M, elt)
}

func (this *StringSet) Clear() {
	this.M = make(map[string]struct{})
}

func (this *StringSet) ToSlice() []string {
	count := len(this.M)
	if count == 0 {
		return []string{}
	}

	r := make([]string, count)

	i := 0
	for elt := range this.M {
		r[i] = elt
		i++
	}

	return r
}
