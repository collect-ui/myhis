package plugins

import (
	"encoding/json"
	common "github.com/collect-ui/collect/src/collect/common"
	config "github.com/collect-ui/collect/src/collect/config"
	templateService "github.com/collect-ui/collect/src/collect/service_imp"
	utils "github.com/collect-ui/collect/src/collect/utils"
	//"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"log"
	//"net/http"
	"time"
	//"unicode/utf8"
)

type ShellTerm struct {
	templateService.BaseHandler
}

func updateTimer(timer *time.Timer) {
	timer.Reset(time.Hour * 4)
}

type MessageData struct {
	Type string
	Rows int
	Cols int
	Data string
}

func (si *ShellTerm) HandlerData(template *config.Template, handlerParam *config.HandlerParam, ts *templateService.TemplateService) *common.Result {
	conn := ts.GetThirdData(SshName).(*SSHConnection)
	var result interface{}

	session, error := conn.NewSession()
	if error != nil {
		return common.NotOk(error.Error())
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("linux", 80, 40, modes); err != nil {
		log.Fatal("request for pseudo terminal failed: ", err)
	}
	c := ts.GetWs()
	wsOutput := ts.GetWsOutput(template.GetParams())
	session.Stdout = &wsOutput
	session.Stderr = &wsOutput
	pipeIn, _ := session.StdinPipe()
	defer func() {
		// 输入通道关闭
		pipeIn.Close()
		//连接回话关闭
		session.Close()
		// ssh 客户端关闭
		conn.Close()
	}()
	session.Shell()
	// 如果30分钟内容不输入，直接关闭
	timer := time.NewTimer(time.Hour * 4)
	updateTime := utils.Debounce(func() {
		updateTimer(timer)

	}, 5)
	stopCh := make(chan struct{}, 1)
	go func() {
		for {
			_, data, err := c.ReadMessage()
			messageData := MessageData{}
			json.Unmarshal(data, &messageData)
			if err != nil {
				log.Println("read:", err)
				stopCh <- struct{}{}
				break
			}
			updateTime()
			if messageData.Type == "text" {
				// 添加日志

				_, err = pipeIn.Write([]byte(messageData.Data))

			} else if messageData.Type == "resize" {

				session.WindowChange(messageData.Rows, messageData.Cols)
			} else if messageData.Type == "pwd" {
				// 发送带有特殊标志符的 pwd 命令
				_, err = pipeIn.Write([]byte("echo '<<<PWD_START>>>'; pwd; echo '<<<PWD_END>>>';\n"))
				if err != nil {
					log.Println("failed to write pwd command to stdin: ", err)
					stopCh <- struct{}{}
					break
				}

			}

			if err != nil {
				log.Println(err)
				log.Println("执行结束")
				stopCh <- struct{}{}
				break
			}
		}
	}()
	// 如果停止了、或者超时了，直接返回
	for {
		select {
		case <-stopCh:
			return common.Ok(result, "结束执行")
		case <-timer.C:
			wsOutput.Write([]byte("\n\n\t由于长时间没有操作,自动结束执行"))
			return common.Ok(result, "执行超时")
		}
	}
}
