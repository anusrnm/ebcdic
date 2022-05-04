package ebcdic
import (
	"encoding/hex"
	"fmt"
	"testing"
)
func TestDumper(t *testing.T) {
	raw := "C1c2C3F1f2"
	tb, _ := hex.DecodeString(raw)
	fmt.Println("%V",tb)
	fmt.Println(Dump(tb))
}