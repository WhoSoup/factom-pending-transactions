package main

import (
	"fmt"
	"log"
	"time"

	"github.com/FactomProject/factom"
)

type List struct {
	height      int64
	has         map[string]bool
	items       []Entry
	disappeared []Entry
}

func newList(h int64) *List {
	l := new(List)
	l.height = h
	l.has = make(map[string]bool)
	return l
}

type Entry struct {
	Seen        time.Time
	Gone        time.Time
	Transaction factom.PendingTransaction
}

func (l *List) Add(p []factom.PendingTransaction) {
	for _, t := range p {
		if l.has[t.TxID] {
			continue
		}
		l.has[t.TxID] = true
		l.items = append(l.items, Entry{Seen: time.Now(), Transaction: t})
		fmt.Println("added new tx", t.TxID)
	}

	for _, e := range l.items {
		if notin(e, p) {
			fmt.Println("tx", e, "disappeared")
			e.Gone = time.Now()
			l.disappeared = append(l.disappeared, e)
		}
	}
}

func notin(e Entry, p []factom.PendingTransaction) bool {
	for _, t := range p {
		if e.Transaction.TxID == t.TxID {
			return false
		}
	}
	return true
}

var checkHeight int64 = 0
var lists map[int64]*List

func getList(h int64) *List {
	if list, ok := lists[h]; ok {
		return list
	}
	lists[h] = newList(h)
	return lists[h]
}

func poll() {
	height, err := factom.GetHeights()
	if err != nil {
		log.Println(err)
		return
	}

	if height.DirectoryBlockHeight > checkHeight {
		checkHeight = height.DirectoryBlockHeight
		if list, ok := lists[height.DirectoryBlockHeight]; ok {
			compareWithBlock(list)
		}
	}

	pending, err := factom.GetPendingTransactions()
	if err != nil {
		log.Println(err)
		return
	}

	getList(height.LeaderHeight + 1).Add(pending)
}

func compareWithBlock(l *List) {
	fb, _, err := factom.GetFBlockByHeight(l.height)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Printf("======= Analysis block %d ===========\n", l.height)
	for _, tjs := range fb.Transactions {
		if l.has[tjs.TxID] {
			delete(l.has, tjs.TxID)
		} else {
			fmt.Println("pending list never got transaction:", tjs)
		}
	}

	if len(l.has) > 0 {
		fmt.Println("Pendings that never showed up in the block:")
		for _, p := range l.items {
			if l.has[p.Transaction.TxID] {
				fmt.Println(p)
			}
		}
	}
	fmt.Println("====================")
}

func main() {
	lists = make(map[int64]*List)
	factom.SetFactomdServer("spoon:8088")
	//factom.EnableCookies()

	timer := time.NewTicker(time.Millisecond * 250)
	for range timer.C {
		poll()
	}

}
