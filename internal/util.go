package internal

import "os"

func MustGetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic("no environment variable found for " + key)
	}
	return val
}
