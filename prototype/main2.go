package main

import (
	"fmt"
	"strings"
)

func filter[T any](ss []T, test func(T) bool) (ret []T) {
    for _, s := range ss {
        if test(s) {
            ret = append(ret, s)
        }
    }
    return
}

func main(){
	ss := []string{"foo_1", "asdf", "loooooooong", "nfoo_1", "foo_2"}

	mytest := func(s string) bool { return !strings.HasPrefix(s, "foo_") && len(s) <= 7 }
	s2 := filter(ss, mytest)

	fmt.Println(s2)
}