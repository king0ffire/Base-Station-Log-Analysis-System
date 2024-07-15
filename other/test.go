package test

import (
	"encoding/json"
	"fmt"
)

// 定义一个结构体
type Person struct {
	Name string
	Age  int
	City string
}

func test() {
	// 创建一个Person结构体实例
	p := Person{Name: "John Doe", Age: 30, City: "New York"}

	// 将结构体转换为JSON字节切片
	jsonBytes, err := json.Marshal(p)
	if err != nil {
		fmt.Println("Error marshaling:", err)
		return
	}
	fmt.Println("JSON:", jsonBytes)
	// 将JSON字节切片转换为map[string]interface{}
	var data map[string]interface{}
	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		fmt.Println("Error unmarshaling:", err)
		return
	}

	// 打印转换后的字典
	fmt.Println("JSON as dictionary:", data)
}
