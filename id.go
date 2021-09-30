package idgen

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
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

// IDGenerator id generator interface
type IDGenerator interface {
	NextID() (int64, error)            // deprecated
	NextIDs(size int) ([]int64, error) // deprecated
	ParseID(id int64) string

	ID() int64
	IDs(size int) []int64
}

// Config configuration
type Config struct {
	DataCenterID int64
	WorkerID     int64
}

type generator struct {
	mutex *sync.Mutex

	lastTimestamp int64
	datacenterID  int64
	workerID      int64
	sequence      int64
}

// NewIDGenerator new id generator instance
func NewIDGenerator(dataCenterID, workerID int64) (IDGenerator, error) {
	if dataCenterID > maxDataCenterID || dataCenterID < 0 {
		return nil, fmt.Errorf("data center id should be greater than 0 and less than %d", maxDataCenterID)
	}
	if workerID > maxWorkerID || workerID < 0 {
		return nil, fmt.Errorf("worker id should be greater than 0 and less than %d", maxWorkerID)
	}

	gen := new(generator)

	gen.mutex = new(sync.Mutex)
	gen.lastTimestamp = -1
	gen.datacenterID = dataCenterID
	gen.workerID = workerID
	gen.sequence = int64(0)

	return gen, nil
}

// NewIDGeneratorByConfig new id generator instance by config
func NewIDGeneratorByConfig(config Config) (IDGenerator, error) {
	return NewIDGenerator(config.DataCenterID, config.WorkerID)
}

// MustNewIDGenerator new id generator instance without error
func MustNewIDGenerator(config Config) IDGenerator {
	gen, err := NewIDGenerator(config.DataCenterID, config.WorkerID)
	if err != nil {
		panic(err)
	}

	return gen
}

func (gen *generator) NextID() (int64, error) {
	gen.mutex.Lock()
	defer gen.mutex.Unlock()

	timestamp := time.Now().UnixNano() / nanosInMilli
	delta := gen.lastTimestamp - timestamp
	if delta > 0 {
		return -1, fmt.Errorf("clock moved backwards, refusing to generate id for %d milliseconds", delta)
	}

	if delta == 0 {
		gen.sequence = (gen.sequence + 1) & sequenceMask
		if gen.sequence == 0 {
			time.Sleep(1 * time.Millisecond) // until next millisecond
			timestamp = time.Now().UnixNano() / nanosInMilli
		}
	} else {
		gen.sequence = int64(0)
	}

	gen.lastTimestamp = timestamp
	// fmt.Println(timestamp, gen.datacenterID, gen.workerID, gen.sequence)
	return (timestamp-mcepoch)<<timestampLeftShift |
		gen.datacenterID<<dataCenterIDLeftShift |
		gen.workerID<<workerIDLeftShift |
		gen.sequence, nil
}

func (gen *generator) NextIDs(size int) ([]int64, error) {
	result := make([]int64, size)
	var resultErr error

	for i := 0; i < size; i++ {
		id, err := gen.NextID()
		if err != nil {
			if resultErr == nil {
				resultErr = err
			} else {
				resultErr = errors.Wrapf(resultErr, "%v", err)
			}
		} else {
			result[i] = id
		}
	}

	return result, resultErr
}

func (gen *generator) ParseID(id int64) string {
	timestamp := (id>>timestampLeftShift)&0x1FFFFFFFFFF + mcepoch
	dataCenterID := (id >> dataCenterIDLeftShift) & 0x1F
	workerID := (id >> workerIDLeftShift) & 0x1F
	sequence := id & 0xFFF

	return fmt.Sprintf("timestamp: %d, data center id: %d, worker id: %d, sequence: %d",
		timestamp, dataCenterID, workerID, sequence)
}

func (gen *generator) ID() int64 {
	id, err := gen.NextID()
	if err != nil {
		panic(err)
	}

	return id
}

func (gen *generator) IDs(size int) []int64 {
	ids, err := gen.NextIDs(size)
	if err != nil {
		panic(err)
	}

	return ids
}
