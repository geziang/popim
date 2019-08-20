package cache

type IMMsgInStore struct {
	Timestamp int64
	FromId uint
	ToId uint
	Content string
}