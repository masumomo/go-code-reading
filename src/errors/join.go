// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// Join returns nil if errs contains no non-nil values.
// The error formats as the concatenation of the strings obtained
// by calling the Error method of each element of errs, with a newline
// between each string.
// JOINができるようになってる！！！！！
func Join(errs ...error) error {
	// len(errs)じゃダメなんか？ by Miki
	// return err とかをJOINしたもののうち、nilのものは除きたいんやな by Hikari
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{
		errs: make([]error, 0, n), // 一旦これしたいんだね by Miki
	}
	// 2回ループ回してでもサイズ固定した方がええんやなやっぱ(nが大きいとそうなる？)　いや違うか。キャパシティを正確にしたいからか by Hikari
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

type joinError struct {
	errs []error
}

func (e *joinError) Error() string {
	var b []byte
	for i, err := range e.errs {
		// 一個だったら改行したくないとかそういう感じか
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, err.Error()...)
	}
	return string(b)
}

func (e *joinError) Unwrap() []error {
	return e.errs
}
