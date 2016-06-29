package udpchat

import (
	"testing"
)

func TestSend(t *testing.T) {
	c, _ := NewClient("wutao")
	c.SendFile("testfile.txt")
}
