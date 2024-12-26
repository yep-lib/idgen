package idgen

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	sequenceBit     = 12 // 12 bit sequence number
	workerIDBit     = 5  // 5 bit worker id
	dataCenterIDBit = 5  // 5 bit data center id

	sequenceMask    = -1 ^ (-1 << sequenceBit)
	maxWorkerID     = -1 ^ (-1 << workerIDBit)
	maxDataCenterID = -1 ^ (-1 << dataCenterIDBit)

	workerIDLeftShift     = sequenceBit                                 // 12 bit
	dataCenterIDLeftShift = sequenceBit + workerIDBit                   // 17 bit
	timestampLeftShift    = sequenceBit + workerIDBit + dataCenterIDBit // 22 bit

	mcEpoch = int64(1531272738938) // YEP epoch
)

var (
	mutex *sync.Mutex

	datacenterID int64
	workerID     int64

	lastTimestamp int64
	sequence      int64
)

func init() {
	mutex = new(sync.Mutex)

	// init datacenter id
	strDataCenterID := os.Getenv("IDGEN_DATACENTER_ID")
	if strDataCenterID == "" {
		datacenterID = rand.Int63()
	} else {
		datacenterID, _ = strconv.ParseInt(strDataCenterID, 10, 64)
	}
	if datacenterID > maxDataCenterID {
		datacenterID = maxDataCenterID
	}
	if datacenterID < 0 {
		datacenterID = 0
	}

	// init worker id
	strWorkerID := os.Getenv("IDGEN_WORKER_ID")
	if strWorkerID == "" {
		workerID = rand.Int63()
	} else {
		workerID, _ = strconv.ParseInt(strWorkerID, 10, 64)
	}
	if workerID > maxWorkerID {
		workerID = maxWorkerID
	}
	if workerID < 0 {
		workerID = 0
	}

	lastTimestamp = -1
	sequence = int64(0)
}

func NextIntID() int64 {
	mutex.Lock()
	defer mutex.Unlock()

	timestamp := time.Now().UnixMilli()
	delta := lastTimestamp - timestamp
	if delta > 0 {
		panic(fmt.Errorf(
			"clock moved backwards, refusing to generate id for %d milliseconds",
			delta,
		))
	}

	if delta == 0 {
		sequence = (sequence + 1) & sequenceMask
		if sequence == 0 {
			time.Sleep(1 * time.Millisecond) // until next millisecond
			timestamp = time.Now().UnixMilli()
		}
	} else {
		sequence = int64(0)
	}

	lastTimestamp = timestamp

	return (timestamp-mcEpoch)<<timestampLeftShift |
		datacenterID<<dataCenterIDLeftShift |
		workerID<<workerIDLeftShift |
		sequence
}

func NextStringID() string {
	intID := NextIntID()
	strID := fmt.Sprintf("%x", intID)

	return strID
}

func ParseIntID(id int64) string {
	timestamp := (id>>timestampLeftShift)&0x1FFFFFFFFFF + mcEpoch
	dataCenterID := (id >> dataCenterIDLeftShift) & 0x1F
	workerID := (id >> workerIDLeftShift) & 0x1F
	sequence := id & 0xFFF

	return fmt.Sprintf("timestamp: %d, data center id: %d, worker id: %d, sequence: %d",
		timestamp, dataCenterID, workerID, sequence)
}

func ParseStringID(s string) string {
	id, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		panic(err)
	}

	timestamp := (id>>timestampLeftShift)&0x1FFFFFFFFFF + mcEpoch
	dataCenterID := (id >> dataCenterIDLeftShift) & 0x1F
	workerID := (id >> workerIDLeftShift) & 0x1F
	sequence := id & 0xFFF

	return fmt.Sprintf("timestamp: %d, data center id: %d, worker id: %d, sequence: %d",
		timestamp, dataCenterID, workerID, sequence)
}
