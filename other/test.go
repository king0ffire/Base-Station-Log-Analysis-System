package main

import (
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	s := os.Getenv("SESSION_KEY")
	fmt.Println(s)
	fmt.Println(base64.Encoding)
}
