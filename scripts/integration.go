package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	fc_open20210406 "github.com/alibabacloud-go/fc-open-20210406/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

func CreateClient(accessKey, secretKey, accountId, fcRegion string) (_result *fc_open20210406.Client, _err error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(secretKey),
	}
	// 访问的域名
	config.Endpoint = tea.String(fmt.Sprintf("%s.%s.fc.aliyuncs.com", accountId, fcRegion))
	_result = &fc_open20210406.Client{}
	_result, _err = fc_open20210406.NewClient(config)
	return _result, _err
}

func _getIdleFunction(client *fc_open20210406.Client, serviceName string) (_functionName string, _err error) {
	listFunctionsHeaders := &fc_open20210406.ListFunctionsHeaders{}
	listFunctionsRequest := &fc_open20210406.ListFunctionsRequest{}
	runtime := &util.RuntimeOptions{}
	functionNames := make([]string, 0)
	_response, _err := client.ListFunctionsWithOptions(tea.String(serviceName), listFunctionsRequest, listFunctionsHeaders, runtime)
	if _err != nil {
		return "", _err
	}
	for _, fc := range _response.Body.Functions {
		functionNames = append(functionNames, *fc.FunctionName)
	}

	for _, functionName := range functionNames {
		listStatefulAsyncInvocationsHeaders := &fc_open20210406.ListStatefulAsyncInvocationsHeaders{}
		listStatefulAsyncInvocationsRequest := &fc_open20210406.ListStatefulAsyncInvocationsRequest{}
		_response, _err := client.ListStatefulAsyncInvocationsWithOptions(tea.String(serviceName), tea.String(functionName), listStatefulAsyncInvocationsRequest, listStatefulAsyncInvocationsHeaders, runtime)
		if _err != nil {
			return "", _err
		}
		idle := true
		for _, invocation := range _response.Body.Invocations {
			if fmt.Sprint(*invocation.EndTime) == "0" {
				idle = false
				break
			}
		}
		if idle {
			return functionName, nil
		}
	}
	return "", nil
}

func _checkInvocationIsExist(client *fc_open20210406.Client, serviceName, invocationId string) (exist bool, _functionName string, _err error) {
	listFunctionsHeaders := &fc_open20210406.ListFunctionsHeaders{}
	listFunctionsRequest := &fc_open20210406.ListFunctionsRequest{}
	runtime := &util.RuntimeOptions{}
	_response, _err := client.ListFunctionsWithOptions(tea.String(serviceName), listFunctionsRequest, listFunctionsHeaders, runtime)
	if _err != nil {
		return false, "", _err
	}

	var returnError error
	for _, fc := range _response.Body.Functions {
		functionName := *fc.FunctionName
		getStatefulAsyncInvocationHeaders := &fc_open20210406.GetStatefulAsyncInvocationHeaders{}
		getStatefulAsyncInvocationRequest := &fc_open20210406.GetStatefulAsyncInvocationRequest{}
		_resp, _err := client.GetStatefulAsyncInvocationWithOptions(tea.String(serviceName), tea.String(functionName), tea.String(invocationId), getStatefulAsyncInvocationRequest, getStatefulAsyncInvocationHeaders, runtime)
		if _err != nil {
			if strings.Contains(_err.Error(), "StatefulAsyncInvocationNotFound") {
				continue
			}
			returnError = _err
			log.Printf("[ERROR] getting invocation %s failed. Error:%v", invocationId, _err)
		} else if *_resp.Body.InvocationId == invocationId {
			return true, functionName, nil
		}
	}
	return false, "", returnError
}

func _invokeFunction(client *fc_open20210406.Client, serviceName, functionName, invocationId, ossBucketRegion, ossBucketName, ossObjectPath, diffFuncNames string) (_err error) {
	invokeFunctionHeaders := &fc_open20210406.InvokeFunctionHeaders{
		XFcInvocationType:            tea.String("Async"),
		XFcLogType:                   tea.String("None"),
		XFcStatefulAsyncInvocationId: tea.String(invocationId),
	}
	body := map[string]interface{}{
		"diffFuncNames":   diffFuncNames,
		"ossBucketName":   ossBucketName,
		"ossBucketRegion": ossBucketRegion,
		"ossObjectPath":   ossObjectPath,
	}
	bodyString, err := json.Marshal(body)
	if err != nil {
		return err
	}
	invokeFunctionRequest := &fc_open20210406.InvokeFunctionRequest{
		Qualifier: tea.String("LATEST"),
		Body:      util.ToBytes(tea.String(string(bodyString))),
	}
	runtime := &util.RuntimeOptions{}

	_, _err = client.InvokeFunctionWithOptions(tea.String(serviceName), tea.String(functionName), invokeFunctionRequest, invokeFunctionHeaders, runtime)
	if _err != nil {
		if strings.Contains(_err.Error(), "StatefulAsyncInvocationAlreadyExists") {
			log.Printf("the invocation %s has been existed in the function: %s", invocationId, functionName)
		} else {
			return _err
		}
	}

	getStatefulAsyncInvocationHeaders := &fc_open20210406.GetStatefulAsyncInvocationHeaders{}
	getStatefulAsyncInvocationRequest := &fc_open20210406.GetStatefulAsyncInvocationRequest{}

	for true {
		_response, _err := client.GetStatefulAsyncInvocationWithOptions(tea.String(serviceName), tea.String(functionName), tea.String(invocationId), getStatefulAsyncInvocationRequest, getStatefulAsyncInvocationHeaders, runtime)
		if _err != nil {
			return _err
		}
		if fmt.Sprint(*_response.Body.EndTime) == "0" {
			time.Sleep(5 * time.Second)
			continue
		}
		if *_response.Body.Status != "Succeeded" {
			return fmt.Errorf(*_response.Body.InvocationErrorMessage)
		}
		return nil
	}
	return nil
}
func main() {
	accessKey := os.Args[1]
	secretKey := os.Args[2]
	accountId := os.Args[3]
	serviceName := os.Args[4]
	fcRegion := os.Args[5]
	client, _err := CreateClient(accessKey, secretKey, accountId, fcRegion)
	if _err != nil {
		log.Println(_err)
		os.Exit(1)
	}
	ossBucketRegion := strings.TrimSpace(os.Args[6])
	ossBucketName := strings.TrimSpace(os.Args[7])
	ossObjectPath := strings.TrimSpace(os.Args[8])
	invocationId := strings.Replace(ossObjectPath, "/", "_", -1)
	diffFuncNames := strings.Trim(strings.TrimSpace(os.Args[9]), ";")
	functionName := ""
	log.Println("trace id:", invocationId)
	if exist, fcName, err := _checkInvocationIsExist(client, serviceName, invocationId); err != nil {
		log.Printf("[ERROR] checking invocation %s failed. Error: %v", invocationId, err)
	} else if exist {
		log.Printf("the invocation %s has been existed in the function %s", invocationId, fcName)
		os.Exit(0)
	}
	for true {
		if idleFunc, err := _getIdleFunction(client, serviceName); err != nil {
			log.Println("_getIdleFunction got an error:", err)
			os.Exit(1)
		} else if idleFunc != "" {
			functionName = idleFunc
			break
		}
	}

	log.Println("using function", functionName)
	if err := _invokeFunction(client, serviceName, functionName, invocationId, ossBucketRegion, ossBucketName, ossObjectPath, diffFuncNames); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
