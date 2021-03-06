package server

import (
	"time"
	"log"
	"github.com/jc3wish/Bifrost/plugin"
	pluginDriver "github.com/jc3wish/Bifrost/plugin/driver"
	"runtime"
	"fmt"
	"runtime/debug"
)

func (This *ToServer) pluginClose(){
	if This.PluginConnKey != ""{
		plugin.Close(This.ToServerKey,This.PluginConnKey)
	}
	This.PluginConn = nil
}

func (This *ToServer) consume_to_server(db *db,SchemaName string,TableName string) {
	toServerPositionBinlogKey := getToServerBinlogkey(db,This)
	defer func() {
		This.pluginClose()
		if err := recover();err !=nil{
			log.Println(db.Name,"SchemaName:",SchemaName,"TableName:",TableName, This.PluginName,This.ToServerKey,"ToServer consume_to_server over;err:",err,"debug",string(debug.Stack()))
			return
		}else{
			log.Println(db.Name,"SchemaName:",SchemaName,"TableName:",TableName, This.PluginName,This.ToServerKey,"ToServer consume_to_server over")
		}
	}()
	log.Println(db.Name,"SchemaName:",SchemaName,"TableName:",TableName, This.PluginName,This.ToServerKey,"ToServer consume_to_server  start")
	c := This.ToServerChan.To

	var data *pluginDriver.PluginDataType
	CheckStatusFun := func(){
		if db.killStatus == 1{
			runtime.Goexit()
		}
		if This.Status == "deling"{
			This.Status = "deled"
			delBinlogPosition(toServerPositionBinlogKey)
			runtime.Goexit()
		}
	}
	var PluginBinlog *pluginDriver.PluginBinlog
	var errs error
	binlogKey := getToServerBinlogkey(db,This)

	SaveBinlog := func(){
		if PluginBinlog != nil {
			db.Lock()
			This.BinlogFileNum = PluginBinlog.BinlogFileNum
			This.BinlogPosition = PluginBinlog.BinlogPosition
			db.Unlock()
			//这里保存位点是为了刷到磁盘,这个位点在重启 配置文件恢复的时候，会根据最小的 ToServerList 的位点进行自动替换
			saveBinlogPosition(binlogKey, PluginBinlog.BinlogFileNum, PluginBinlog.BinlogPosition)
		}
	}
	var fordo int = 0
	var lastErrId int = 0
	for {
		CheckStatusFun()
		select {
		case data = <- c:
			CheckStatusFun()
			fordo = 0
			lastErrId = 0
			for {
				errs = nil
				PluginBinlog,errs = This.sendToServer(data)
				if This.MustBeSuccess == true {
					if errs == nil{
						if lastErrId > 0 {
							This.DelWaitError()
							lastErrId = 0
						}
						break
					} else {
						if lastErrId > 0{
							dealStatus := This.GetWaitErrorDeal()
							if dealStatus == -1{
								lastErrId = 0
								break
							}
							if dealStatus == 1{
								This.DelWaitError()
								lastErrId = 0
								break
							}
						}else{
							This.AddWaitError(errs,data)
							lastErrId = 1
						}
					}
					fordo++
					if fordo==3{
						CheckStatusFun()
						time.Sleep(2 * time.Second)
						fordo = 0
					}
				}else{
					PluginBinlog = &pluginDriver.PluginBinlog{data.BinlogFileNum,data.BinlogPosition}
					break
				}
			}
			//这里保存位点，为是了显示的时候，可以直接从内存中读取
			SaveBinlog()
			break
		case <-time.After(5 * time.Second):
			PluginBinlog,_ = This.PluginConn.Commit()
			SaveBinlog()
			break
		}
	}
}

func (This *ToServer) filterField(data *pluginDriver.PluginDataType) bool{
	n := len(data.Rows)
	if n == 0{
		return true
	}
	if len(This.FieldList) == 0{
		return true
	}

	if n == 1 {
		m := make(map[string]interface{})
		for _, key := range This.FieldList {
			if _, ok := data.Rows[0][key]; ok {
				m[key] = data.Rows[0][key]
			}
		}
		data.Rows[0] = m
	}else{
		m_before := make(map[string]interface{})
		m_after := make(map[string]interface{})
		var isNotUpdate bool = true
		for _, key := range This.FieldList {
			if _, ok := data.Rows[0][key]; ok {
				m_before[key] = data.Rows[0][key]
				m_after[key] = data.Rows[1][key]
				if m_before[key] != m_after[key]{
					isNotUpdate = false
				}
			}
		}
		//假如所有字段内容都未变更，并且过滤了这个功能，则直接返回false
		if isNotUpdate && This.FilterUpdate{
			return  false
		}
		data.Rows[0] = m_before
		data.Rows[1] = m_after
	}
	return true
}

func (This *ToServer) sendToServer(data *pluginDriver.PluginDataType) ( Binlog *pluginDriver.PluginBinlog,err error){
	defer func() {
		if err2 := recover();err2!=nil{
			err = fmt.Errorf(This.ToServerKey,string(debug.Stack()))
			log.Println(This.ToServerKey,"sendToServer err:",err)
			func() {
				defer func() {
					if err2 := recover();err2!=nil{
						return
					}
				}()
				This.PluginConn.Close()
			}()
			This.PluginConn.Connect()
		}
	}()
	if This.PluginConn == nil{
		This.PluginConn,This.PluginConnKey = plugin.Start(This.ToServerKey)
		if This.PluginConn == nil{
			err = fmt.Errorf("Plugin:"+This.PluginName+" ToServerKey:"+ This.ToServerKey+ " start err,return nil")
			return Binlog,err
		}
		err := This.PluginConn.SetParam(This.PluginParam)
		if err != nil{
			return Binlog,err
		}
	}

	// 只有所有字段内容都没有更新，并且开启了过滤功能的情况下，才会返回false
	if This.filterField(data) == false{
		return &pluginDriver.PluginBinlog{data.BinlogFileNum,data.BinlogPosition},nil
	}

	switch data.EventType {
	case "insert":
		Binlog, err = This.PluginConn.Insert(data)
		break
	case "update":
		Binlog, err = This.PluginConn.Update(data)
		break
	case "delete":
		Binlog, err = This.PluginConn.Del(data)
		break
	case "sql":
		Binlog, err = This.PluginConn.Query(data)
		break
	default:
		break
	}
	return
}

