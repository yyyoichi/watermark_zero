package strmark

type Mark interface {
	Encode(src string) (mark []bool, err error)
	Decode(mark []bool) (src string, err error)
}
