package template

import (
	"fmt"
	"testing"
)

func Test_getKind(t *testing.T) {
	fmt.Println(getKind([]string{"1", "2"}))
}
