package mail

import (
	"io"
)

// 76个字符换行
type writer struct {
	i int
	w io.Writer
}

func newWriter(w io.Writer) *writer {
	return &writer{
		w: w,
	}
}

// 写, 76个字符换行
func (self *writer) Write(p []byte) (n int, err error) {
	w := self.w
	r := 76 - self.i
	for len(p) > 0 {
		m := 0
		k := 0
		if len(p) <= r {
			k = len(p)
		} else {
			k = r
		}
		m, err = w.Write(p[:k])
		n += m
		self.i += m
		p = p[m:]
		if err != nil {
			return
		}

		if self.i == 76 {
			self.i = 0
			_, err = w.Write([]byte("\r\n"))
			if err != nil {
				return
			}
		}
		r = 76 - self.i
	}
	return
}
