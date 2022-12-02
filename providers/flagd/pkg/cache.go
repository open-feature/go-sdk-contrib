package flagd

type Cache[K comparable, V any] interface {
	Add(K, V) (evicted bool)
	Purge()
	Get(K) (value V, ok bool)
	Remove(K) (present bool)
}
