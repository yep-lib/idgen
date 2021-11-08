package idgen

import (
	"fmt"
	"math/rand"
	"os"
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

	mcepoch = int64(1531272738938) // YEP epoch
)

var nanosInMilli = time.Millisecond.Nanoseconds()
var generatorInstance *generator

type generator struct {
	mutex *sync.Mutex

	lastTimestamp int64
	datacenterID  int64
	workerID      int64
	sequence      int64
}

func init() {
	var dataCenterID, workerID int64
	strDataCenterID := os.Getenv("DATACENTER_ID")
	if strDataCenterID == "" {
		dataCenterID = rand.Int63n(maxDataCenterID)
	}
	strWorkerID := os.Getenv("WORKER_ID")
	if strWorkerID == "" {
		workerID = rand.Int63n(maxWorkerID)
	}

	generatorInstance = &generator{
		mutex:         &sync.Mutex{},
		lastTimestamp: -1,
		datacenterID:  dataCenterID,
		workerID:      workerID,
		sequence:      0,
	}
}

func NewID() int64 {
	generatorInstance.mutex.Lock()
	defer generatorInstance.mutex.Unlock()

	timestamp := time.Now().UnixNano() / nanosInMilli
	delta := generatorInstance.lastTimestamp - timestamp
	if delta > 0 {
		panic(fmt.Errorf("clock moved backwards, refusing to generate id for %d milliseconds", delta))
	}

	if delta == 0 {
		generatorInstance.sequence = (generatorInstance.sequence + 1) & sequenceMask
		if generatorInstance.sequence == 0 {
			time.Sleep(1 * time.Millisecond) // until next millisecond
			timestamp = time.Now().UnixNano() / nanosInMilli
		}
	} else {
		generatorInstance.sequence = int64(0)
	}

	generatorInstance.lastTimestamp = timestamp

	return (timestamp-mcepoch)<<timestampLeftShift |
		generatorInstance.datacenterID<<dataCenterIDLeftShift |
		generatorInstance.workerID<<workerIDLeftShift |
		generatorInstance.sequence
}

func NewIDs(size int) []int64 {
	result := make([]int64, size)

	for i := 0; i < size; i++ {
		result[i] = NewID()
	}

	return result
}

func ParseID(id int64) string {
	timestamp := (id>>timestampLeftShift)&0x1FFFFFFFFFF + mcepoch
	dataCenterID := (id >> dataCenterIDLeftShift) & 0x1F
	workerID := (id >> workerIDLeftShift) & 0x1F
	sequence := id & 0xFFF

	return fmt.Sprintf("timestamp: %d, data center id: %d, worker id: %d, sequence: %d",
		timestamp, dataCenterID, workerID, sequence)
}
