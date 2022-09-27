package hello

import (
	"strings"
	"testing"
)

func TestLength(t *testing.T) {
	msg := SayHello("World")
	length := len(msg)
	if length != 12 {
		t.Errorf("SayHello(\"World\") length is %d; want 12", length)
	}
}

func TestContainsUTF(t *testing.T) {
	msg := SayHello("嗨")
	if !strings.Contains(msg, "嗨") {
		t.Error("SayHello(\"嗨\") doesn't support UTF8")
	}
}
