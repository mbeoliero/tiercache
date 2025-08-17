package tiercache

import "sync"

var optionsPool = sync.Pool{
	New: func() interface{} {
		return &cacheOpts{}
	},
}

type cacheOpts struct {
	getLevel int
}

type OptFunc func(*cacheOpts)

func WithGetLevel(level int) OptFunc {
	return func(opts *cacheOpts) {
		opts.getLevel = level
	}
}

func defaultOpts() *cacheOpts {
	opt := optionsPool.Get().(*cacheOpts)
	opt.free()
	return opt
}

func (m *cacheOpts) free() {
	m.getLevel = 0
}
