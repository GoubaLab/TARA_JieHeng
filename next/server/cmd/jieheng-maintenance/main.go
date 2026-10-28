package main

import (
	"context"
	"flag"
	"fmt"
	"jieheng.local/tara/internal/assurance"
	"os"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	action := flag.String("action", "verify", "backup / verify / restore")
	source := flag.String("source", "", "数据库文件（backup）或备份目录（verify/restore）")
	destination := flag.String("destination", "", "尚不存在的目标目录（backup/restore）")
	flag.Parse()
	if *source == "" {
		return fmt.Errorf("请提供 --source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var err error
	switch *action {
	case "backup":
		if *destination == "" {
			return fmt.Errorf("请提供 --destination")
		}
		err = assurance.BackupDatabase(ctx, *source, *destination)
	case "verify":
		err = assurance.VerifyBackup(ctx, *source)
	case "restore":
		if *destination == "" {
			return fmt.Errorf("请提供 --destination")
		}
		err = assurance.RestoreBackup(ctx, *source, *destination)
	default:
		return fmt.Errorf("不支持的操作")
	}
	if err == nil {
		fmt.Println("操作完成。还原不会替换运行中的数据库；切换服务前请核验还原目录。")
	}
	return err
}
