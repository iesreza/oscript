package gen

import (
	"github.com/getevo/evo/lib/gpath"
	"github.com/getevo/evo/lib/log"
	"os"
	"os/exec"
	"testing"
)

func TestOpenAndGenerate(t *testing.T) {
	generated, err := OpenAndGenerate("../gorm-test/sample.oscript")
	if err != nil {
		panic(err)
	}

	f, err := gpath.Open("../gorm-test/models.go")
	if err != nil {
		panic(err)
	}
	f.WriteString(generated)
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "../gorm-test"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		log.Error(err)
	}
}
