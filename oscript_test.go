package oscript

import (
	"fmt"
	"testing"
)

func TestOpenAndParse(t *testing.T) {
	var ctx, err = OpenAndParse("./sample/example.oscript")
	fmt.Println(err)
	fmt.Println(PrettyStruct(ctx))
}
