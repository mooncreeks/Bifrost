function transferBool(b){
    if(b=="true" || b == "1"){
        return true;
    }else{
        return false;
    }
}
function doGetPluginParam(){
	var result = {data:{},status:false,msg:"error"}
    var data = {};

	var QueueName = $("#ActiveMQ_Queue").val();
    var Expir = $("#ActiveMQ_Plugin_Contair input[name='Expir']").val();
    var Persistent = transferBool($("#RabbitMQ_Plugin_Contair #RabbitMQ_Persistent").val());

    if (Expir != "" && Expir != null && isNaN(Expir)){
        result.msg = "Expir must be int!"
        return result;
    }

	data["QueueName"] = QueueName;
	data["Persistent"] = Persistent;
	data["Expir"] = parseInt(Expir);

	result.data = data;
	result.msg = "success";
	result.status = true;
    return result;
}
