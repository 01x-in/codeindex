package main

import (
	"fmt"

	"example.com/testproject/pkg/service"
)

func main() {
	svc := service.NewUserService()
	user := svc.GetUser("abc123")
	fmt.Println(user.Name)
}
