package rworker

import "sync"

func Job(wg *sync.WaitGroup, fn func() error, rate chan struct{}, errCh chan<- error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		rate <- struct{}{}
		if err := fn(); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
		<-rate
	}()
}
