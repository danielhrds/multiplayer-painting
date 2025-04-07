package main

import (
	"fmt"
	"sync"
)

func test() {
	m := map[int]int{}

	mux := &sync.RWMutex{}

	go writeLoop(m, mux)
	go readLoop(m, mux)
	go readLoop(m, mux)
	go readLoop(m, mux)
	go readLoop(m, mux)

	// stop program from exiting, must be killed
	// block := make(chan struct{})
	// <-block
	v := 0
	for {
		v++
	}
}

func writeLoop(m map[int]int, mux *sync.RWMutex) {
	for i := 0; i < 100; i++ {
		mux.Lock()
		m[i] = i
		mux.Unlock()
	}
}

func readLoop(m map[int]int, mux *sync.RWMutex) {
	mux.RLock()
	for k, v := range m {
		fmt.Println(k, "-", v)
	}
	mux.RUnlock()

}

// package main

// import (
// 	"fmt"
// 	"sync"
// )

// func main() {
// 	m := map[int]int{}

// 	mux := &sync.Mutex{}

// 	go writeLoop(m, mux)
// 	go readLoop(m, mux)

// 	// stop program from exiting, must be killed
// 	block := make(chan struct{})
// 	<-block
// }

// func writeLoop(m map[int]int, mux *sync.Mutex) {
// 	for {
// 		for i := 0; i < 100; i++ {
// 			mux.Lock()
// 			m[i] = i
// 			mux.Unlock()
// 		}
// 	}
// }

// func readLoop(m map[int]int, mux *sync.Mutex) {
// 	for {
// 		mux.Lock()
// 		for k, v := range m {
// 			fmt.Println(k, "-", v)
// 		}
// 		mux.Unlock()
// 	}
// }
