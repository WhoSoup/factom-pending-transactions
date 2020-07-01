package main

import (
	"fmt"
	"log"
	"os"
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

func (l *List) Add(t factom.PendingTransaction) {
	if l.has[t.TxID] {
		return
	}
	l.has[t.TxID] = true
	l.items = append(l.items, Entry{Seen: time.Now(), Transaction: t})
	fmt.Println("added new tx", l.height, t.TxID, time.Now())
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
	fmt.Fprintf(f, "%+v\n", pending)

	for _, t := range pending {
		getList(int64(t.DBHeight)).Add(t)
	}
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

var f *os.File

func main() {
	lists = make(map[int64]*List)
	factom.SetFactomdServer("localhost:8088")
	//factom.EnableCookies()

	f, _ = os.Create("abc.txt")
	defer f.Close()

	timer := time.NewTicker(time.Millisecond * 100)
	for range timer.C {
		poll()
	}

}
