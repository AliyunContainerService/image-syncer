package concurrent

import sync2 "sync"

// CreateRoutinesAndWaitForFinish creates num of goroutines which all execute f() and wait for them to finish.
func CreateRoutinesAndWaitForFinish(num int, f func()) {
	wg := sync2.WaitGroup{}
	// open c.routineNum goroutines and wait
	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}
