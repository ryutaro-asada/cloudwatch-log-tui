package main

import (
    "bytes"
    "sync"
    "testing"
)

var bufPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

const dataSize = 1024

// ---------- バッファを都度生成 ----------
func BenchmarkBufferNoPool(b *testing.B) {
    for i := 0; i < b.N; i++ {
        buf := new(bytes.Buffer)
        buf.Grow(dataSize)
        buf.Write(make([]byte, dataSize))
        _ = buf.Bytes()
    }
}

// ---------- sync.Pool を使う ----------
func BenchmarkBufferWithPool(b *testing.B) {
    for i := 0; i < b.N; i++ {
        buf := bufPool.Get().(*bytes.Buffer)
        buf.Reset() // 再利用する場合はリセットが重要
        buf.Grow(dataSize)
        buf.Write(make([]byte, dataSize))
        _ = buf.Bytes()
        bufPool.Put(buf)
    }
}
