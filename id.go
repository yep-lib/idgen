package idgen

import (
	"fmt"
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

var nanosInMilli = time.Millisecond.Nanoseconds()

// IDGenerator id generator interface
type IDGenerator interface {
	MustNextIntID() int64
	MustNextStringID() string

	ParseIntID(id int64) string
	ParseStringID(id string) string
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
func NewIDGenerator(dataCenterID, workerID int64) (*generator, error) {
	if dataCenterID > maxDataCenterID || dataCenterID < 0 {
		return nil, fmt.Errorf("data center id should be greater than 0 and less than %d", maxDataCenterID)
	}
	if workerID > maxWorkerID || workerID < 0 {
		return nil, fmt.Errorf("worker id should be greater than 0 and less than %d", maxWorkerID)
	}

	g := new(generator)

	g.mutex = new(sync.Mutex)
	g.lastTimestamp = -1
	g.datacenterID = dataCenterID
	g.workerID = workerID
	g.sequence = int64(0)

	return g, nil
}

// NewIDGeneratorByConfig new id generator instance by config
func NewIDGeneratorByConfig(config Config) (*generator, error) {
	return NewIDGenerator(config.DataCenterID, config.WorkerID)
}

// MustNewIDGenerator new id generator instance without error
func MustNewIDGenerator(config Config) *generator {
	g, err := NewIDGenerator(config.DataCenterID, config.WorkerID)
	if err != nil {
		panic(err)
	}

	return g
}

func (g *generator) MustNextIntID() int64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	timestamp := time.Now().UnixMilli()
	delta := g.lastTimestamp - timestamp
	if delta > 0 {
		panic(fmt.Errorf(
			"clock moved backwards, refusing to generate id for %d milliseconds",
			delta,
		))
	}

	if delta == 0 {
		g.sequence = (g.sequence + 1) & sequenceMask
		if g.sequence == 0 {
			time.Sleep(1 * time.Millisecond) // until next millisecond
			timestamp = time.Now().UnixMilli()
		}
	} else {
		g.sequence = int64(0)
	}

	g.lastTimestamp = timestamp
	// fmt.Println(timestamp, g.datacenterID, g.workerID, g.sequence)
	return (timestamp-mcEpoch)<<timestampLeftShift |
		g.datacenterID<<dataCenterIDLeftShift |
		g.workerID<<workerIDLeftShift |
		g.sequence
}

func (g *generator) MustNextStringID() string {
	intID := g.MustNextIntID()
	strID := fmt.Sprintf("%x", intID)

	return strID
}

func (g *generator) ParseIntID(id int64) string {
	timestamp := (id>>timestampLeftShift)&0x1FFFFFFFFFF + mcEpoch
	dataCenterID := (id >> dataCenterIDLeftShift) & 0x1F
	workerID := (id >> workerIDLeftShift) & 0x1F
	sequence := id & 0xFFF

	return fmt.Sprintf("timestamp: %d, data center id: %d, worker id: %d, sequence: %d",
		timestamp, dataCenterID, workerID, sequence)
}

func (g *generator) ParseStringID(s string) string {
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
