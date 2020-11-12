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

// DescribeInstanceAccounts invokes the drds.DescribeInstanceAccounts API synchronously
func (client *Client) DescribeInstanceAccounts(request *DescribeInstanceAccountsRequest) (response *DescribeInstanceAccountsResponse, err error) {
	response = CreateDescribeInstanceAccountsResponse()
	err = client.DoAction(request, response)
	return
}

// DescribeInstanceAccountsWithChan invokes the drds.DescribeInstanceAccounts API asynchronously
func (client *Client) DescribeInstanceAccountsWithChan(request *DescribeInstanceAccountsRequest) (<-chan *DescribeInstanceAccountsResponse, <-chan error) {
	responseChan := make(chan *DescribeInstanceAccountsResponse, 1)
	errChan := make(chan error, 1)
	err := client.AddAsyncTask(func() {
		defer close(responseChan)
		defer close(errChan)
		response, err := client.DescribeInstanceAccounts(request)
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

// DescribeInstanceAccountsWithCallback invokes the drds.DescribeInstanceAccounts API asynchronously
func (client *Client) DescribeInstanceAccountsWithCallback(request *DescribeInstanceAccountsRequest, callback func(response *DescribeInstanceAccountsResponse, err error)) <-chan int {
	result := make(chan int, 1)
	err := client.AddAsyncTask(func() {
		var response *DescribeInstanceAccountsResponse
		var err error
		defer close(result)
		response, err = client.DescribeInstanceAccounts(request)
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

// DescribeInstanceAccountsRequest is the request struct for api DescribeInstanceAccounts
type DescribeInstanceAccountsRequest struct {
	*requests.RpcRequest
	DrdsInstanceId string `position:"Query" name:"DrdsInstanceId"`
}

// DescribeInstanceAccountsResponse is the response struct for api DescribeInstanceAccounts
type DescribeInstanceAccountsResponse struct {
	*responses.BaseResponse
	RequestId        string           `json:"RequestId" xml:"RequestId"`
	Success          bool             `json:"Success" xml:"Success"`
	InstanceAccounts InstanceAccounts `json:"InstanceAccounts" xml:"InstanceAccounts"`
}

// CreateDescribeInstanceAccountsRequest creates a request to invoke DescribeInstanceAccounts API
func CreateDescribeInstanceAccountsRequest() (request *DescribeInstanceAccountsRequest) {
	request = &DescribeInstanceAccountsRequest{
		RpcRequest: &requests.RpcRequest{},
	}
	request.InitWithApiInfo("Drds", "2019-01-23", "DescribeInstanceAccounts", "Drds", "openAPI")
	request.Method = requests.POST
	return
}

// CreateDescribeInstanceAccountsResponse creates a response to parse from DescribeInstanceAccounts response
func CreateDescribeInstanceAccountsResponse() (response *DescribeInstanceAccountsResponse) {
	response = &DescribeInstanceAccountsResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}
