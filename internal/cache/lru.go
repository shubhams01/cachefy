package cache

import "container/list"

type lruEntry struct {
	key   string
	entry Entry
}

type lru struct {
	capacity int
	items    map[string]*list.Element
	list     *list.List
}

func newLRU(capacity int) *lru {
	if capacity <= 0 {
		capacity = 1
	}

	return &lru{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (l *lru) get(key string) (Entry, bool) {
	element, ok := l.items[key]

	if !ok {
		return Entry{}, false
	}

	l.list.MoveToFront(element)

	item := element.Value.(*lruEntry)

	return item.entry, true
}

func (l *lru) set(key string, entry Entry) bool {
	if element, ok := l.items[key]; ok {
		item := element.Value.(*lruEntry)

		item.entry = entry

		l.list.MoveToFront(element)

		return false
	}

	element := l.list.PushFront(&lruEntry{
		key:   key,
		entry: entry,
	})

	l.items[key] = element

	if l.list.Len() > l.capacity {
		l.removeOldest()
		return true
	}

	return false
}

func (l *lru) delete(key string) bool {
	element, ok := l.items[key]

	if !ok {
		return false
	}

	delete(l.items, key)
	l.list.Remove(element)

	return true
}

func (l *lru) removeOldest() {
	element := l.list.Back()

	if element == nil {
		return
	}

	item := element.Value.(*lruEntry)

	delete(l.items, item.key)

	l.list.Remove(element)
}

func (l *lru) clear() {
	l.items = make(map[string]*list.Element)
	l.list.Init()
}

func (l *lru) len() int {
	return l.list.Len()
}
