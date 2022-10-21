package handle_kvstore

import (
	"github.com/slawomir-pryczek/handler_socket2/hscommon"
	"sync"
)

type item struct {
	data    []byte
	expires int

	is_sensitive bool
}

type pool struct {
	mu   sync.RWMutex
	data map[string]item
}

const pool_count = uint32(10)

var pools [pool_count]*pool

// fnv1 inspired hashing algo
func fasthash(s string) uint32 {
	prime32 := uint32(16777619)
	h := uint32(0)
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * prime32
	}
	return h
}

func _getpool(k string) uint32 {
	return (fasthash(k) % pool_count)
}

func KeySet(k string, data []byte, ttl int, is_sensitive bool) {
	n := item{}
	n.data = data
	n.is_sensitive = is_sensitive
	if ttl > 0 {
		n.expires = hscommon.TSNow() + ttl
	}

	poolno := _getpool(k)
	pools[poolno].mu.Lock()
	pools[poolno].data[k] = n
	pools[poolno].mu.Unlock()
}

func KeyGet(k string, def []byte) []byte {
	return keyGet(k, def).data
}

func keyGet(k string, def []byte) item {
	poolno := _getpool(k)
	now := hscommon.TSNow()
	i := item{}

	pools[poolno].mu.RLock()
	_tmp := pools[poolno].data[k]
	if now <= _tmp.expires || _tmp.expires == 0 {
		i.expires = _tmp.expires
		i.data = _tmp.data
		i.is_sensitive = _tmp.is_sensitive
	} else {
		i.expires = 0
		i.data = def
		i.is_sensitive = false
	}
	pools[poolno].mu.RUnlock()

	if i.data != nil {
		ret := make([]byte, len(i.data))
		copy(ret, i.data)
		i.data = ret
	}
	return i
}
