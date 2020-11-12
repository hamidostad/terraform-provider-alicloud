package drds

//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
// Code generated by Alibaba Cloud SDK Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

// UpdateInstanceNetwork invokes the drds.UpdateInstanceNetwork API synchronously
func (client *Client) UpdateInstanceNetwork(request *UpdateInstanceNetworkRequest) (response *UpdateInstanceNetworkResponse, err error) {
	response = CreateUpdateInstanceNetworkResponse()
	err = client.DoAction(request, response)
	return
}

// UpdateInstanceNetworkWithChan invokes the drds.UpdateInstanceNetwork API asynchronously
func (client *Client) UpdateInstanceNetworkWithChan(request *UpdateInstanceNetworkRequest) (<-chan *UpdateInstanceNetworkResponse, <-chan error) {
	responseChan := make(chan *UpdateInstanceNetworkResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.UpdateInstanceNetwork(request)
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response
		}
	})
	if err != nil {
		errChan <- err
		close(responseChan)
		close(errChan)
	}
	return responseChan, errChan
}

// UpdateInstanceNetworkWithCallback invokes the drds.UpdateInstanceNetwork API asynchronously
func (client *Client) UpdateInstanceNetworkWithCallback(request *UpdateInstanceNetworkRequest, callback func(response *UpdateInstanceNetworkResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *UpdateInstanceNetworkResponse
		var err error
		defer close(result)
		response, err = client.UpdateInstanceNetwork(request)
		callback(response, err)
		result <- 1
	})
	if err != nil {
		defer close(result)
		callback(nil, err)
		result <- 0
	}
	return result
}

// UpdateInstanceNetworkRequest is the request struct for api UpdateInstanceNetwork
type UpdateInstanceNetworkRequest struct {
	*requests.RpcRequest
	DrdsInstanceId         string           `position:"Query" name:"DrdsInstanceId"`
	RetainClassic          requests.Boolean `position:"Query" name:"RetainClassic"`
	ClassicExpiredDays     requests.Integer `position:"Query" name:"ClassicExpiredDays"`
	SrcInstanceNetworkType string           `position:"Query" name:"SrcInstanceNetworkType"`
}

// UpdateInstanceNetworkResponse is the response struct for api UpdateInstanceNetwork
type UpdateInstanceNetworkResponse struct {
	*responses.BaseResponse
	RequestId string `json:"RequestId" xml:"RequestId"`
	Success   bool   `json:"Success" xml:"Success"`
}

// CreateUpdateInstanceNetworkRequest creates a request to invoke UpdateInstanceNetwork API
func CreateUpdateInstanceNetworkRequest() (request *UpdateInstanceNetworkRequest) {
	request = &UpdateInstanceNetworkRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Drds", "2019-01-23", "UpdateInstanceNetwork", "Drds", "openAPI")
	request.Method = requests.POST
	return
}

// CreateUpdateInstanceNetworkResponse creates a response to parse from UpdateInstanceNetwork response
func CreateUpdateInstanceNetworkResponse() (response *UpdateInstanceNetworkResponse) {
	response = &UpdateInstanceNetworkResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
