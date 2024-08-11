package utils

import (
	"context"
	_ "embed"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/chromedp/chromedp"
	"github.com/yzbtdiy/inspection_assistant/models"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// 使用go embed嵌入powershell脚本,由于go暂时没有合适的kdbx库,所以使用powershell脚本进行kdbx操作,需要调用keepass.exe
//
//go:embed get_entry.ps1
var kpScript string

// 全局变量,站点索引计数器
var curIndex int = 0

// 全局变量,临时脚本文件名
var tmpScriptName string

// golang控制台输入不显示密码: https://www.golang.cx/go/golang%E6%8E%A7%E5%88%B6%E5%8F%B0%E8%BE%93%E5%85%A5%E5%AF%86%E7%A0%81%E4%B8%8D%E6%98%BE%E7%A4%BA.html
func HideInput() (string, error) {
	// 禁用输入回显
	oldState, _ := term.MakeRaw(int(syscall.Stdin))
	defer term.Restore(int(syscall.Stdin), oldState)
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	password := string(passwordBytes)
	return password, nil
}

// 读取配置文件
func GetConfig() (config models.Config) {
	yamlFile, err := os.Open("./config.yaml")
	if err != nil {
		log.Fatal("读取配置文件失败,请检查当前路径是否存在config.yaml文件")
	}
	defer yamlFile.Close()
	decoder := yaml.NewDecoder(yamlFile)

	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("配置文件解析失败,请检查配置", err)
	}
	return config
}

// 生成脚本文件,调用powershell脚本需要传入参数,所以生成临时脚本文件,执行结束后清除
func GenScript() {
	tmp, err := os.CreateTemp(".", "*.ps1")
	if err != nil {
		log.Fatal(err)
	}
	// defer os.Remove(tmp.Name())
	tmp.WriteString(kpScript)
	tmp.Close()
	tmpScriptName = tmp.Name()
}

func CleanScript() {
	os.Remove(tmpScriptName)
}

func DbValidSucc(config models.Config, kdbxPass string) {
	GenScript()
	command := exec.Command("powershell", "-ExecutionPolicy", "ByPass", " -File", tmpScriptName, config.KeeppassPath, config.KpdbPath, kdbxPass)
	output, _ := command.CombinedOutput()
	if string(output) == "CON_DB_SUCC\r\n" {
		log.Println("密码验证通过,启动浏览器执行代填任务")
		// 若脚本输出LOAD_KEEPASS_ERR,说明没有找到keepass.exe文件
	} else if string(output) == "LOAD_KEEPASS_ERR\r\n" {
		CleanScript()
		log.Fatal("加载keepass.exe失败,请检路径是否正确")
		//  若脚本输出DB_VALID_ERR,说明数据库接入失败
	} else if string(output) == "DB_VALID_ERR\r\n" {
		CleanScript()
		log.Fatal("数据库验证失败,请检查密码或kxdb路径")
		// 其他结果输出到日志
	} else {
		CleanScript()
		log.Fatal(string(output))
	}
}

func GetEntry(config models.Config, kdbxPass, entryTitle string) (entry models.Entry) {
	// 执行powershell, 脚本接收keepass路径,kdbx数据库路径,kdbx数据库密码,kdbx数据库中记录的站点标题
	command := exec.Command("powershell", "-ExecutionPolicy", "ByPass", " -File", tmpScriptName, config.KeeppassPath, config.KpdbPath, kdbxPass, entryTitle)
	output, _ := command.Output()
	// 脚本正常执行输出站点url,用户,密码
	if len(strings.Split(string(output), "\r\n")) == 4 {
		entry = models.Entry{Url: strings.Split(string(output), "\r\n")[0], Username: strings.Split(string(output), "\r\n")[1], Password: strings.Split(string(output), "\r\n")[2]}
		// 若脚本输出GET_ENTRY_ERR,表示根据配置文件entry_title没有查找到对应记录
	} else if string(output) == "GET_ENTRY_ERR\r\n" {
		CleanScript()
		log.Fatal("获取站点信息失败,请检查配置文件entry_title")
		// 其他结果输出到日志
	} else {
		CleanScript()
		log.Fatal(string(output))
	}
	return entry
}

// 打开站点,填充账号密码,递归函数,由于chromedp打开新标签需要上一个页面的context,所以使用递归实现遍历站点信息
func AutoFill(ctx context.Context, config models.Config, kdbxPass string) {
	entry := GetEntry(config, kdbxPass, config.SitesInfo[curIndex].EntryTitle)
	curCtx, _ := chromedp.NewContext(ctx)
	// defer cancel()
	// 华为USG6630E特殊处理,密码输入框点击后会生成新的输入框,使用pass_fill_locator作为选择器输入内容
	var passInputLocator string
	if config.SitesInfo[curIndex].PassFillLocator != "" {
		passInputLocator = config.SitesInfo[curIndex].PassFillLocator
	} else {
		passInputLocator = config.SitesInfo[curIndex].PassLocator
	}
	// 创建tasks任务列表
	var tasks chromedp.Tasks = chromedp.Tasks{
		// 跳转到目标页面
		chromedp.Navigate(entry.Url),
		// 填充用户名
		chromedp.Click(config.SitesInfo[curIndex].UserLocator, chromedp.NodeVisible),
		chromedp.SendKeys(config.SitesInfo[curIndex].UserLocator, entry.Username),
		// 填充密码
		chromedp.Click(config.SitesInfo[curIndex].PassLocator, chromedp.NodeVisible),
		chromedp.SendKeys(passInputLocator, entry.Password),
	}
	// 如果auto_login为true,则点击登录按钮
	if config.SitesInfo[curIndex].AutoLogin {
		tasks = append(tasks, chromedp.Click(config.SitesInfo[curIndex].LoginLocator, chromedp.NodeVisible))
	}
	// 执行tasks任务列表
	err := chromedp.Run(curCtx, tasks)
	if err != nil {
		if err == context.Canceled {
			CleanScript()
			log.Fatal("检测到浏览器关闭,退出")
		} else {
			CleanScript()
			// log.Println(err)
			log.Fatal("浏览器接入失败,若浏览器正在运行请关闭浏览器重试")
		}
	}
	log.Println("访问 ", config.SitesInfo[curIndex].EntryTitle, " 并填入账号和密码")
	// 索引计数器加1
	curIndex++
	if curIndex < len(config.SitesInfo) {
		AutoFill(curCtx, config, kdbxPass)
	}
}
