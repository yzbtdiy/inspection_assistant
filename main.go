package main

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
	"github.com/yzbtdiy/inspection_assistant/models"
	"github.com/yzbtdiy/inspection_assistant/utils"
)

func main() {
	// 读取配置文件
	var config models.Config = utils.GetConfig()
	// 读取密码,不显示输入内容
	log.Println("请输入Keepass数据库密码: ")
	kdbxPass, _ := utils.HideInput()

	// 验证密码是否正确
	utils.DbValidSucc(config, kdbxPass)

	// chrome自定义运行参数
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(config.BrowserPath),
		chromedp.UserDataDir(config.BrowserData),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.IgnoreCertErrors,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.Flag("start-maximized", true),
		// chromedp.Flag("enable-automation", false),
		// chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// chromedp.Flag("ignore-certificate-errors", true),
	)

	// 创建Context
	allocCtx, _ := chromedp.NewExecAllocator(
		context.Background(),
		options...,
	)
	// defer cancel()

	// 设置超时
	// ctx, cancel := context.WithTimeout(allocCtx, 15*time.Second)
	// defer cancel()

	//自动填充登录信息
	utils.AutoFill(allocCtx, config, kdbxPass)
	log.Println("执行完毕,自动退出,浏览器将继续运行")
	// 清除临时脚本文件
	utils.CleanScript()
}
