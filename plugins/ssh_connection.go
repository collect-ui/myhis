package plugins

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

// SSHConnection 包装 SSH 客户端连接，包含进程标识符
// 用于在单个请求流程中共享 SSH 连接，通过 TemplateService.thirdData 存储
// 嵌入 *ssh.Client 确保与现有代码的兼容性
type SSHConnection struct {
	*ssh.Client        // 嵌入 SSH 客户端，保持现有方法调用兼容
	ProcessID   string // 进程标识符，用于会话跟踪和调试
}

// NewSSHConnection 创建新的 SSHConnection 实例
// client: 已建立的 SSH 客户端连接
// processID: 进程标识符，如果为空则自动生成 UUID
func NewSSHConnection(client *ssh.Client, processID string) *SSHConnection {
	if processID == "" {
		processID = uuid.New().String()
	}
	return &SSHConnection{
		Client:    client,
		ProcessID: processID,
	}
}

// GetProcessID 获取进程标识符
func (c *SSHConnection) GetProcessID() string {
	return c.ProcessID
}

// SetProcessID 设置进程标识符
func (c *SSHConnection) SetProcessID(processID string) {
	c.ProcessID = processID
}
