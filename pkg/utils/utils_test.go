package utils

import (
	"fmt"
	"testing"
)

func TestCountPassageWord(t *testing.T) {
	passageLetter := "English Test, word and letter."
	words_1 := CountPassageWord(passageLetter)
	if words_1 != 7 {
		t.Log(fmt.Sprintf("words_1: %v", words_1))
		t.Fail()
	}

	passage := `
	this is a passage to test the function.
	这是一个测试函数的文章。
	`
	words := CountPassageWord(passage)

	if words != 21 {
		t.Log(fmt.Sprint("count words: ", words))
		t.Fail()
	}
}
