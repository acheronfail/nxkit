package tools

import (
	"context"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/nsz"
)

func CompressNSZ(path string, keyset keys.Keys) (string, error) {
	return CompressNSZWithProgress(path, keyset, nil)
}

func CompressNSZWithProgress(path string, keyset keys.Keys, progress ProgressFunc) (string, error) {
	return CompressNSZWithProgressContext(context.Background(), path, keyset, progress)
}

func CompressNSZWithProgressContext(ctx context.Context, path string, keyset keys.Keys, progress ProgressFunc) (string, error) {
	output, err := nsz.CompressedPath(path)
	if err != nil {
		return "", err
	}
	return output, nsz.CompressNSPWithProgressContext(ctx, path, output, keyset, nsz.ProgressFunc(progress))
}

func DecompressNSZ(path string) (string, error) {
	return DecompressNSZWithProgress(path, nil)
}

func DecompressNSZWithProgress(path string, progress ProgressFunc) (string, error) {
	return DecompressNSZWithProgressContext(context.Background(), path, progress)
}

func DecompressNSZWithProgressContext(ctx context.Context, path string, progress ProgressFunc) (string, error) {
	output, err := nsz.DecompressedPath(path)
	if err != nil {
		return "", err
	}
	return output, nsz.DecompressNSZWithProgressContext(ctx, path, output, nsz.ProgressFunc(progress))
}
