// https://gitlab.com/imqksl/rangelist
/*
Copyright 2022 guanyupeng

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package rangelist

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
