package sliceutils

import (
	"github.com/samber/lo"
	"strings"
)

func ToSet[K comparable](coll []K) map[K]struct{} {
	return lo.SliceToMap(coll, func(item K) (K, struct{}) {
		return item, struct{}{}
	})
}

func FlatMapArgs(coll []string) []string {
	return lo.FlatMap(coll, func(item string, _ int) []string {
		return strings.Split(item, ",")
	})
}

func FlatMapArgsToSet(coll []string) map[string]struct{} {
	return ToSet(FlatMapArgs(coll))
}
