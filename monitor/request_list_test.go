package monitor

import (
	"testing"
	"time"

	"github.com/0xPolygonHermez/zkevm-pool-manager/types"
)

func TestRequestListDelete(t *testing.T) {
	el := newMonitorRequestList()

	now := time.Now()
	past := now.Add(-time.Minute * 5)
	future := now.Add(time.Minute * 5)

	el.add(&monitorRequest{
		l2Tx:      types.L2Transaction{Id: 1, Hash: "0x01"},
		nextRetry: now,
	})
	el.add(&monitorRequest{
		l2Tx:      types.L2Transaction{Id: 2, Hash: "0x02"},
		nextRetry: now,
	})
	el.add(&monitorRequest{
		l2Tx:      types.L2Transaction{Id: 3, Hash: "0x03"},
		nextRetry: past,
	})
	el.add(&monitorRequest{
		l2Tx:      types.L2Transaction{Id: 4, Hash: "0x04"},
		nextRetry: past,
	})
	el.add(&monitorRequest{
		l2Tx:      types.L2Transaction{Id: 5, Hash: "0x05"},
		nextRetry: future,
	})

	sort := []string{"0x03", "0x04", "0x01", "0x02", "0x05"}

	for index, req := range el.sorted {
		if sort[index] != req.l2Tx.Hash {
			t.Fatalf("Sort error. Expected %s, Actual %s", sort[index], req.l2Tx.Hash)
		}
	}

	delreqs := []string{"0x01", "0x04", "0x05"}

	for _, delreq := range delreqs {
		count := el.len()
		el.delete(&monitorRequest{l2Tx: types.L2Transaction{Hash: delreq}})

		for i := 0; i < el.len(); i++ {
			if el.getByIndex(i).l2Tx.Hash == delreq {
				t.Fatalf("Delete error. %s req was not deleted", delreq)
			}
		}

		if el.len() != count-1 {
			t.Fatalf("Length error. Length %d. Expected %d", el.len(), count)
		}
	}

	if el.delete(&monitorRequest{l2Tx: types.L2Transaction{Hash: "0x05"}}) {
		t.Fatal("Delete error. 0x05 req was deleted and should not exist in the list")
	}
}
