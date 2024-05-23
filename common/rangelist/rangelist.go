package rangelist

// https://gitlab.com/imqksl/rangelist, MIT License

func NewRangeList() *RangeList {
	return &RangeList{
		list: make([]struct {
			minVal int
			maxVal int
		}, 0),
	}
}

type RangeList struct {
	list []struct {
		minVal int
		maxVal int
	}
}

func (r *RangeList) Add(minVal, maxVal int) {
	left, right := 0, len(r.list)-1
	for left < len(r.list) && r.list[left].maxVal < minVal {
		left++
	}
	for right >= 0 && r.list[right].minVal > maxVal {
		right--
	}
	if left <= right {
		minVal = min(minVal, r.list[left].minVal)
		maxVal = max(maxVal, r.list[right].maxVal)
	}
	r.list = append(r.list[:left], append([]struct {
		minVal int
		maxVal int
	}{{minVal, maxVal}}, r.list[right+1:]...)...)
}

func (r *RangeList) In(minVal, maxVal int) bool {
	if len(r.list) == 0 {
		return false
	}
	i, j := 0, len(r.list)-1
	for i <= j {
		m := i + (j-i)/2
		switch {
		case r.list[m].minVal >= maxVal:
			j = m - 1
		case r.list[m].maxVal <= minVal:
			i = m + 1
		default:
			return r.list[m].minVal <= minVal && r.list[m].maxVal >= maxVal
		}
	}
	return false
}
