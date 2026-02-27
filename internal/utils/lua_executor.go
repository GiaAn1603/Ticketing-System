package utils

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
)

func EvalShaWithFallback(ctx context.Context, rdb *redis.Client, sha string, scriptBody string, keys []string, args ...interface{}) *redis.Cmd {
	cmd := rdb.EvalSha(ctx, sha, keys, args...)
	if err := cmd.Err(); err != nil && strings.HasPrefix(err.Error(), "NOSCRIPT") {
		return rdb.Eval(ctx, scriptBody, keys, args...)
	}

	return cmd
}
