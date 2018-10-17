package inmemory

import (
	"fmt"
	"testing"
)

func TestCreateInMemoryConfig(t *testing.T) {
	cfg := ConfigInMemory{}
	cfg.Init()
	fmt.Printf("message need to print,%+v\n", cfg)
}
