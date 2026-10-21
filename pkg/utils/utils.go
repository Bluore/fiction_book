package utils

import (
	"crypto/md5"
	"encoding/binary"
	"unicode"
)

// CountPassageWord todo 获取文章中字词数量
func CountPassageWord(passage string) int {
	count := 0
	lastIsLetter := false

	for _, letter := range passage {
		//fmt.Print(i)
		//fmt.Printf("-%c ", letter)
		// 如果是无意义字符则跳过
		if unicode.IsSpace(letter) {
			lastIsLetter = false
			continue
		}

		// 如果是英文单词则按照space来分割
		if IsEnglishLetter(letter) {
			if !lastIsLetter {
				count++
			}
			lastIsLetter = true
			continue
		}

		// 如果不是字母则单字算为一个单词
		count++
		lastIsLetter = false

	}

	return count
}

func IsEnglishLetter(letter rune) bool {
	if letter >= 'A' && letter <= 'Z' {
		return true
	}
	if letter >= 'a' && letter <= 'z' {
		return true
	}
	if letter >= '0' && letter <= '9' {
		return true
	}
	return false
}

func HashToInt(in string, numRange uint64) int {
	if numRange <= 0 {
		return 0
	}
	hashByte := md5.Sum([]byte(in))
	hashHigh := binary.BigEndian.Uint64(hashByte[:8])
	hashLow := binary.BigEndian.Uint64(hashByte[8:16])

	return int((hashLow + hashHigh) % numRange)
}
