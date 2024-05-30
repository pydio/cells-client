package rest

import (
	"fmt"
	"github.com/gosuri/uiprogress"
	"io"
	"time"
)

type BarsPool struct {
	*uiprogress.Progress
	showGlobal bool
	nodesBar   *uiprogress.Bar
}

func NewBarsPool(showGlobal bool, totalNodes int, refreshInterval time.Duration) *BarsPool {
	b := &BarsPool{}
	b.Progress = uiprogress.New()
	b.Progress.SetRefreshInterval(refreshInterval)
	b.showGlobal = showGlobal
	if showGlobal { // we are transferring more than one file
		b.nodesBar = b.AddBar(totalNodes)
		b.nodesBar.PrependCompleted()
		b.nodesBar.AppendFunc(func(b *uiprogress.Bar) string {

			if b.Current() == b.Total {
				//return fmt.Sprintf("Transferred %d/%d files and folders in %s.", b.Current(), b.Total, b.TimeElapsedString())
				return fmt.Sprintf("Done in %s.", b.TimeElapsedString())
			} else {
				return fmt.Sprintf("Copying folders and files since %s: %d/%d", b.TimeElapsedString(), b.Current(), b.Total)
			}
		})
	}
	return b
}

func (b *BarsPool) Done() {
	if !b.showGlobal {
		return
	}
	b.nodesBar.Incr()
	if b.nodesBar.Current() == b.nodesBar.Total {
		// Finished, remove all bars
		b.Bars = []*uiprogress.Bar{b.nodesBar}
	}
}

func (b *BarsPool) Get(i int, total int, name string) *uiprogress.Bar {
	idx := i % PoolSize
	var nBars []*uiprogress.Bar
	if b.showGlobal {
		idx++
		nBars = append(nBars, b.nodesBar)
	}
	// Remove old bar
	for k, bar := range b.Bars {
		if k == idx || (b.showGlobal && bar == b.nodesBar) {
			continue
		}
		nBars = append(nBars, bar)
	}
	b.Bars = nBars
	bar := b.AddBar(total)
	bar.PrependCompleted()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprint(name)
	})
	return bar
}

type ReaderWithProgress struct {
	io.Reader
	io.Seeker
	bar   *uiprogress.Bar
	total int
	read  int

	double bool
	first  bool

	errChan chan error
}

func (r *ReaderWithProgress) CreateErrorChan() (chan error, chan struct{}) {
	errors, done := newErrorChan(r.sendErr)
	r.errChan = errors
	return errors, done
}

//func (r *ReaderWithProgress) CreateErrorChan() (chan error, chan struct{}) {
//	done := make(chan struct{}, 1)
//	r.errChan = make(chan error)
//	go func() {
//		for {
//			select {
//			case e := <-r.errChan:
//				r.sendErr(e)
//			case <-done:
//				close(r.errChan)
//				return
//			}
//		}
//	}()
//	return r.errChan, done
//}

func newErrorChan(handleError func(error)) (chan error, chan struct{}) {
	done := make(chan struct{}, 1)
	errChan := make(chan error)
	go func() {
		for {
			select {
			case e := <-errChan:
				handleError(e)
			case <-done:
				close(errChan)
				return
			}
		}
	}()
	return errChan, done
}

func (r *ReaderWithProgress) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == nil {
		if r.double {
			r.read += n / 2
		} else {
			r.read += n
		}
		r.bar.Set(r.read)
	} else if err == io.EOF {
		if r.double && !r.first {
			r.first = true
			r.bar.Set(r.total / 2)
		} else {
			r.bar.Set(r.total)
		}
	}
	return
}

func (r *ReaderWithProgress) Seek(offset int64, whence int) (int64, error) {
	if r.double && r.first {
		r.read = r.total/2 + int(offset)/2
	} else {
		r.read = int(offset)
	}
	r.bar.Set(r.read)
	return r.Seeker.Seek(offset, whence)
}

func (r *ReaderWithProgress) sendErr(err error) {
	r.bar.AppendFunc(func(b *uiprogress.Bar) string {
		return err.Error()
	})
}
