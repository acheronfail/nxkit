package testutils

const (
	DiskImagePath = "../fixtures/fat16/disk.img"
)

func MapSlice[T any, U any](things []T, f func(T) U) []U {
	mapped := make([]U, len(things))
	for i, thing := range things {
		mapped[i] = f(thing)
	}
	return mapped
}
