package protocol

// RemoteFuncProtocol 远程函数协议
type RemoteFuncProtocol struct{}

func (p *RemoteFuncProtocol) Name() string        { return "remoteFunc" }
func (p *RemoteFuncProtocol) DisplayName() string { return "Remote Function" }
func (p *RemoteFuncProtocol) Description() string { return "调度远程三方节点执行函数" }

func (p *RemoteFuncProtocol) ConfigSchema() ConfigSchema {
	return ConfigSchema{
		Fields: []ConfigField{
			{Name: "funcName", Label: "函数名", Type: "string", Required: true, Description: "远程函数名称"},
			{Name: "timeout", Label: "超时时间(秒)", Type: "number", Required: false, Default: "30"},
		},
	}
}
