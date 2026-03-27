package main

import (
	"fmt"
	"os"

	"github.com/whosm123/WPoster/cmd"
)

func main() {
	app, err := cmd.NewApp()
	if err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Printf("程序运行错误: %v\n", err)
		os.Exit(1)
	}
}
