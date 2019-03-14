package hprose

import (
	"fmt"
	"github.com/hprose/hprose-golang/rpc"
	pluginDriver "github.com/jc3wish/Bifrost/plugin/driver"
	"encoding/json"
)

type Stub struct {
	Check   func() error
	ToList  func(SchemaName string,TableName string,EventType string, data interface{}) (e error)
	Insert  func(SchemaName string,TableName string,data map[string]interface{}) (e error)
	Update  func(SchemaName string,TableName string,data []map[string]interface{}) (e error)
	Delete  func(SchemaName string,TableName string,data map[string]interface{}) (e error)
	Query  func(SchemaName string,TableName string,sql string) (e error)
}

var stub *Stub

type Conn struct {
	Uri    string
	status string
	httpClient *rpc.HTTPClient
	tcpClient *rpc.TCPClient
	defaultClient rpc.Client
	clientType string
	err    error
	p PluginParam
}

type PluginParam struct {
	KeyConfig string
	Expir int
	DataType int
	ValConfig string
	Fields []string
}

func newConn(uri string) *Conn{
	f := &Conn{
		Uri:uri,
		status:"close",
	}
	f.Connect()
	return f
}

func (This *Conn) SetParam(p interface{}) error{
	s,err := json.Marshal(p)
	if err != nil{
		return err
	}
	var param PluginParam
	err2 := json.Unmarshal(s,&param)
	if err2 != nil{
		return err2
	}
	This.p = param
	return nil
}

func (This *Conn) GetConnStatus() string {
	return This.status
}

func (This *Conn) SetConnStatus(status string) {
	This.status = status
}

func checkUriType(uri string) string{
	if uri[0:3] == "tcp"{
		return "tcp"
	}
	if uri[0:4] == "http"{
		return "http"
	}
	return ""
}

func (This *Conn) Connect() bool {
	This.clientType = checkUriType(This.Uri)
	switch This.clientType {
	case "tcp":
		This.tcpClient = rpc.NewTCPClient(This.Uri)
		This.tcpClient.UseService(&stub)
		break
	case "http":
		This.httpClient = rpc.NewHTTPClient(This.Uri)
		This.httpClient.UseService(&stub)
		break
	default:
		This.defaultClient = rpc.NewClient(This.Uri)
		This.defaultClient.UseService(&stub)
		break
	}
	This.status = "running"
	return true
}


func (This *Conn) ReConnect() bool {
	This.Connect()
	return true
}

func (This *Conn) HeartCheck() {
	return
}

func (This *Conn) Close() bool {
	This.clientType = checkUriType(This.Uri)
	switch This.clientType {
	case "tcp":
		This.tcpClient.Close()
		break
	case "http":
		This.httpClient.Close()
		break
	default:
		This.defaultClient.Close()
		break
	}
	return true
}


func (This *Conn) CheckUri() error {
	if This.status != "running"{
		return fmt.Errorf("not connect")
	}
	err := stub.Check()
	switch This.clientType {
	case "tcp":
		This.tcpClient.Close()
		break
	case "http":
		This.httpClient.Close()
		break
	default:
		This.defaultClient.Close()
		break
	}
	return err
}


func (This *Conn) Insert(data *pluginDriver.PluginDataType) (bool,error) {
	err := stub.Insert(data.SchemaName,data.TableName,data.Rows[0])
	if err != nil{
		This.err = err
		return false,err
	}
	return true,nil
}

func (This *Conn) Update(data *pluginDriver.PluginDataType) (bool,error) {
	err := stub.Update(data.SchemaName,data.TableName,data.Rows)
	if err != nil{
		This.err = err
		return false,err
	}
	return true,nil
}


func (This *Conn) Del(data *pluginDriver.PluginDataType) (bool,error) {
	err := stub.Delete(data.SchemaName,data.TableName,data.Rows[0])
	if err != nil{
		This.err = err
		return false,err
	}
	return true,nil
}

func (This *Conn) SendToList(data *pluginDriver.PluginDataType) (bool,error) {
	queryType := data.EventType
	var err error
	if queryType == "sql"{
		err = stub.ToList(data.SchemaName,data.TableName,queryType,data.Query)
	}else{
		err = stub.ToList(data.SchemaName,data.TableName,queryType,data.Rows)
	}
	if err != nil{
		This.err = err
		return false,err
	}
	return true,nil
}

func (This *Conn) Query(data *pluginDriver.PluginDataType) (bool,error) {
	err := stub.Query(data.SchemaName,data.TableName,data.Query)
	if err != nil{
		This.err = err
		return false,err
	}
	return true,nil
}