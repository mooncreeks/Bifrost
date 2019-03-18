package src

import (
	"github.com/jc3wish/Bifrost/plugin/driver"
	"github.com/bradfitz/gomemcache/memcache"
	"fmt"
	"encoding/json"
)

const VERSION  = "v1.1.0"
const BIFROST_VERION = "v1.1.0"

func init(){
	driver.Register("memcache",&MyConn{},VERSION,BIFROST_VERION)
}

type MyConn struct {}

func (MyConn *MyConn) Open(uri string) driver.ConnFun{
	return newConn(uri)
}

func (MyConn *MyConn) GetUriExample() string{
	return "127.0.0.1:11211"
}

func (MyConn *MyConn) CheckUri(uri string) error{
	c:= newConn(uri)
	if c.err != nil{
		return c.err
	}
	c.Close()
	return nil
}

type Conn struct {
	uri    		string
	status 		string
	conn   		*memcache.Client
	err    		error
	p 			PluginParam
}

type PluginParam struct {
	KeyConfig 		string
	Expir 			int32
	DataType 		string
	ValConfig 		string
	AddSchemaName 	bool
	AddTableName 	bool
	AddEventType 	bool
}


func newConn(uri string) *Conn{
	f := &Conn{
		uri:uri,
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

func (This *Conn) Connect() bool {
	This.conn = memcache.New(This.uri)
	if This.conn == nil {
		This.err = fmt.Errorf("memcache New failed",This.uri)
		return false
	}
	This.err = nil
	This.status = "running"
	return true
}

func (This *Conn) ReConnect() bool {
	defer func() {
		if err := recover();err !=nil{
			This.err = fmt.Errorf(fmt.Sprint(err))
		}
	}()
	This.Connect()
	return  true
}

func (This *Conn) HeartCheck() {
	return
}

func (This *Conn) Close() bool {
	return true
}

func (This *Conn) getKeyVal(data *driver.PluginDataType,index int) string {
	return driver.TransfeResult(This.p.KeyConfig,data,index)
}

func (This *Conn) getVal(data *driver.PluginDataType,index int) string {
	return driver.TransfeResult(This.p.ValConfig,data,index)
}

func (This *Conn) Insert(data *driver.PluginDataType) (bool,error) {
	return This.Update(data)
}

func (This *Conn) Update(data *driver.PluginDataType) (bool,error) {
	if This.err != nil {
		This.ReConnect()
	}
	index := len(data.Rows)-1
	Key := This.getKeyVal(data,index)
	var Val []byte
	if This.p.ValConfig != ""{
		Val = []byte(This.getVal(data,index))
	}else{
		p := data.Rows[index]
		if This.p.DataType == "json"{
			if This.p.AddTableName {
				p["TableName"] = data.TableName
			}
			if This.p.AddSchemaName {
				p["SchemaName"] = data.SchemaName
			}
			if This.p.AddEventType {
				p["EventType"] = data.EventType
			}
		}
		Val, _ = json.Marshal(p)
	}
	var err error
	err = This.conn.Set(&memcache.Item{Key: Key, Value: Val,Expiration:This.p.Expir})

	if err != nil {
		This.err = err
		return false,err
	}
	return true,nil
}

func (This *Conn) Del(data *driver.PluginDataType) (bool,error) {
	if This.err != nil {
		This.ReConnect()
	}
	Key := This.getKeyVal(data, 0)
	var err error
	err = This.conn.Delete(Key)
	if err != nil {
		This.err = err
		return false,err
	}
	return true,nil
}

func (This *Conn) Query(data *driver.PluginDataType) (bool,error) {
	return true,nil
}