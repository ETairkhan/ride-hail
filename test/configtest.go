package main

import (
	"fmt"
	"ride-hail/config"
)

func main() {
	defaultValues := config.GetDefaultValue()
	fmt.Println(defaultValues)
}
