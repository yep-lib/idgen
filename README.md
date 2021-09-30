# idgen
ID generator.

A Go version of Twitter snowflake ID generator.

## Installation

```
go get github.com/yep-lib/idgen
```

## Example

```
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/yep-lib/idgen"
)

func main() {
	config := idgen.Config{
		DataCenterID: 1,
		WorkerID:     2,
	}
	idGen := idgen.MustNewIDGenerator(config)

	// asynchronized generate
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			time.Sleep(time.Millisecond * 150)
			id := idGen.ID()
			fmt.Println(id, idGen.ParseID(id))
			wg.Done()
		}()
	}
	wg.Wait()

	// batch generate
	fmt.Println(idGen.NextIDs(6))
}
```
