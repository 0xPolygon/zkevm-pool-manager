package monitor

import (
	"fmt"
	"sort"
	"sync"

	"github.com/0xPolygonHermez/zkevm-pool-manager/log"
)

// monitorRequestList represents a list of monitorRequest indexed by l2Tx.Hash but sorted by nextRetry
type monitorRequestList struct {
	list   map[string]*monitorRequest
	sorted []*monitorRequest
	mutex  sync.Mutex
}

// newMonitorRequestList creates and init an txSortedList
func newMonitorRequestList() *monitorRequestList {
	return &monitorRequestList{
		list:   make(map[string]*monitorRequest),
		sorted: []*monitorRequest{},
	}
}

// add adds a tx to the txSortedList
func (e *monitorRequestList) add(request *monitorRequest) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, found := e.list[request.l2Tx.Hash]; !found {
		e.list[request.l2Tx.Hash] = request
		e.addSort(request)
		return true
	}
	return false
}

// delete deletes the tx from the txSortedList
func (e *monitorRequestList) delete(request *monitorRequest) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if request, found := e.list[request.l2Tx.Hash]; found {
		sLen := len(e.sorted)
		i := sort.Search(sLen, func(i int) bool {
			return e.isGreaterOrEqualThan(e.list[e.sorted[i].l2Tx.Hash], request)
		})

		// i is the index of the first tx that has equal (or lower) nextRetry time than the request. From here we need to go down in the list
		// looking for the sorted[i].l2Tx.Hash equal to request.l2Tx.Hash to get the index of request in the sorted slice.
		// We need to go down until we find the request or we have a request with different (lower) nextRetry time or we reach the end of the list
		for {
			if i == sLen {
				log.Warnf("error deleting monitor request %s from monitoreRequestList, we reach the end of the list", request.l2Tx.Hash)
				return false
			}

			if e.sorted[i].nextRetry != request.nextRetry {
				// we have a request with different (lower) nextRetry time than the request we are looking for, therefore we haven't found the request
				log.Warnf("error deleting monitor request %s from monitoreRequestList, not found in the list of requests with same nextRetry time: %v", request.l2Tx.Hash, request.nextRetry)
				return false
			}

			if e.sorted[i].l2Tx.Hash == request.l2Tx.Hash {
				break
			}

			i = i + 1
		}

		delete(e.list, request.l2Tx.Hash)

		copy(e.sorted[i:], e.sorted[i+1:])
		e.sorted[sLen-1] = nil
		e.sorted = e.sorted[:sLen-1]

		return true
	}
	return false
}

// getByIndex retrieves the monitor request at the i position
func (e *monitorRequestList) getByIndex(i int) *monitorRequest {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	tx := e.sorted[i]

	return tx
}

// len returns the length of the list
func (e *monitorRequestList) len() int {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	l := len(e.sorted)

	return l
}

// print prints the contents of the list
func (e *monitorRequestList) Print() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	fmt.Println("Len: ", len(e.sorted))
	for _, txi := range e.sorted {
		fmt.Printf("Hash=%s, nextRetry=%v\n", txi.l2Tx.Hash, txi.nextRetry)
	}
}

// addSort adds the monitor request to the list in a sorted way
func (e *monitorRequestList) addSort(request *monitorRequest) {
	i := sort.Search(len(e.sorted), func(i int) bool {
		return e.isGreaterThan(e.list[e.sorted[i].l2Tx.Hash], request)
	})

	e.sorted = append(e.sorted, nil)
	copy(e.sorted[i+1:], e.sorted[i:])
	e.sorted[i] = request
	log.Debugf("added monitor request for tx %s with nextRetry time %v to monitorRequestList at index %d from total %d", request.l2Tx.Hash, request.nextRetry, i, len(e.sorted))
}

// isGreaterThan returns true if the request1 has greater nextRetry time than request2
func (e *monitorRequestList) isGreaterThan(request1 *monitorRequest, request2 *monitorRequest) bool {
	return request1.nextRetry.UnixMilli() > request2.nextRetry.UnixMilli()
}

// isGreaterOrEqualThan returns true if the request1 has greater or equal nextRetry time than request2
func (e *monitorRequestList) isGreaterOrEqualThan(request1 *monitorRequest, request2 *monitorRequest) bool {
	return request1.nextRetry.UnixMilli() >= request2.nextRetry.UnixMilli()
}

// GetSorted returns the sorted list of monitor requests
func (e *monitorRequestList) GetSorted() []*monitorRequest {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return e.sorted
}
