package main

import (
	"fmt"
	"ride-hail/internal/common/config"
)

func main() {
	defaultValues := config.GetDefaultValue()
	fmt.Println(defaultValues)
}
