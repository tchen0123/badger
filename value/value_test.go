/*
 * Copyright 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package value

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/badger/y"
)

func TestBasic(t *testing.T) {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "")
	y.Check(err)

	var log Log
	log.Open(dir, "vlog")
	defer log.Close()

	entry := &Entry{
		Key:   []byte("samplekey"),
		Value: []byte("sampleval"),
		Meta:  123,
	}
	ptrs, err := log.Write([]*Entry{entry})
	require.NoError(t, err)
	require.Len(t, ptrs, 1)
	fmt.Printf("Pointer written: %+v", ptrs[0])

	var readEntries []Entry
	e, err := log.Read(ctx, ptrs[0])
	require.NoError(t, err)
	readEntries = append(readEntries, e)
	require.EqualValues(t, []Entry{
		{
			Key:   []byte("samplekey"),
			Value: []byte("sampleval"),
			Meta:  123,
		},
	}, readEntries)
}

func BenchmarkReadWrite(b *testing.B) {
	ctx := context.Background()
	rwRatio := []float32{
		0.1, 0.2, 0.5, 1.0,
	}
	valueSize := []int{
		64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384,
	}

	for _, vsz := range valueSize {
		for _, rw := range rwRatio {
			b.Run(fmt.Sprintf("%3.1f,%04d", rw, vsz), func(b *testing.B) {
				var vl Log
				vl.Open(".", "vlog")
				defer os.Remove("vlog")
				b.ResetTimer()

				b.RunParallel(func(pb *testing.PB) {
					e := new(Entry)
					e.Key = make([]byte, 16)
					e.Value = make([]byte, vsz)

					var ptrs []Pointer

					pt, err := vl.Write([]*Entry{e})
					if err != nil {
						b.Fatalf("Benchmark Write: ", err)
					}
					ptrs = append(ptrs, pt...)

					for pb.Next() {
						f := rand.Float32()
						if f < rw {
							pt, err := vl.Write([]*Entry{e})
							if err != nil {
								b.Fatalf("Benchmark Write: ", err)
							}
							ptrs = append(ptrs, pt...)

						} else {
							ln := len(ptrs)
							if ln == 0 {
								b.Fatalf("Zero length of ptrs")
							}
							idx := rand.Intn(ln)
							e, err := vl.Read(ctx, ptrs[idx])
							if err != nil {
								b.Fatalf("Benchmark Read:", err)
							}
							if len(e.Key) != 16 {
								b.Fatalf("Key is invalid")
							}
							if len(e.Value) != vsz {
								b.Fatalf("Value is invalid")
							}
						}
					}
				})
			})
		}
	}
}
