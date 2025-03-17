package utils

import (
	"sync"
)

// BufferPool 实现内存缓冲区池
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建固定大小的缓冲区池
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get 从池中获取一个缓冲区
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put 将缓冲区放回池中
func (p *BufferPool) Put(buffer []byte) {
	p.pool.Put(buffer)
}
