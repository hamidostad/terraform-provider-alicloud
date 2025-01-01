package alicloud

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/PaesslerAG/jsonpath"
	util "github.com/alibabacloud-go/tea-utils/service"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type AlidnsService struct {
	client *connectivity.AliyunClient
}

func (s *AlidnsService) DescribeAlidnsDomainGroup(id string) (object alidns.DomainGroup, err error) {
	request := alidns.CreateDescribeDomainGroupsRequest()
	request.RegionId = s.client.RegionId

	request.PageNumber = requests.NewInteger(1)
	request.PageSize = requests.NewInteger(20)
	for {

		raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
			return alidnsClient.DescribeDomainGroups(request)
		})
		if err != nil {
			err = WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
			return object, err
		}
		addDebug(request.GetActionName(), raw, request.RpcRequest, request)
		response, _ := raw.(*alidns.DescribeDomainGroupsResponse)

		if len(response.DomainGroups.DomainGroup) < 1 {
			err = WrapErrorf(Error(GetNotFoundMessage("AlidnsDomainGroup", id)), NotFoundMsg, ProviderERROR, response.RequestId)
			return object, err
		}
		for _, object := range response.DomainGroups.DomainGroup {
			if object.GroupId == id {
				return object, nil
			}
		}
		if len(response.DomainGroups.DomainGroup) < PageSizeMedium {
			break
		}
		if page, err := getNextpageNumber(request.PageNumber); err != nil {
			return object, WrapError(err)
		} else {
			request.PageNumber = page
		}
	}
	err = WrapErrorf(Error(GetNotFoundMessage("AlidnsDomainGroup", id)), NotFoundMsg, ProviderERROR)
	return
}

func (s *AlidnsService) DescribeAlidnsRecord(id string) (object alidns.DescribeDomainRecordInfoResponse, err error) {
	request := alidns.CreateDescribeDomainRecordInfoRequest()
	request.RegionId = s.client.RegionId

	request.RecordId = id

	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecordInfo(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"DomainRecordNotBelongToUser", "InvalidRR.NoExist"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("AlidnsRecord", id)), NotFoundMsg, ProviderERROR)
			return
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		return
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	response, _ := raw.(*alidns.DescribeDomainRecordInfoResponse)
	return *response, nil
}

func (s *AlidnsService) ListTagResources(id string) (object alidns.ListTagResourcesResponse, err error) {
	request := alidns.CreateListTagResourcesRequest()
	request.RegionId = s.client.RegionId

	request.ResourceType = "DOMAIN"
	request.ResourceId = &[]string{id}

	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.ListTagResources(request)
	})
	if err != nil {
		err = WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		return
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	response, _ := raw.(*alidns.ListTagResourcesResponse)
	return *response, nil
}

func (s *AlidnsService) SetResourceTags(d *schema.ResourceData, resourceType string) error {
	oldItems, newItems := d.GetChange("tags")
	added := make([]alidns.TagResourcesTag, 0)
	for key, value := range newItems.(map[string]interface{}) {
		added = append(added, alidns.TagResourcesTag{
			Key:   key,
			Value: value.(string),
		})
	}
	removed := make([]string, 0)
	for key := range oldItems.(map[string]interface{}) {
		removed = append(removed, key)
	}
	if len(removed) > 0 {
		request := alidns.CreateUntagResourcesRequest()
		request.RegionId = s.client.RegionId
		request.ResourceId = &[]string{d.Id()}
		request.ResourceType = resourceType
		request.TagKey = &removed
		raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
			return alidnsClient.UntagResources(request)
		})
		addDebug(request.GetActionName(), raw)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
	}
	if len(added) > 0 {
		request := alidns.CreateTagResourcesRequest()
		request.RegionId = s.client.RegionId
		request.ResourceId = &[]string{d.Id()}
		request.ResourceType = resourceType
		request.Tag = &added
		raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
			return alidnsClient.TagResources(request)
		})
		addDebug(request.GetActionName(), raw)
		if err != nil {
			return WrapErrorf(err, DefaultErrorMsg, d.Id(), request.GetActionName(), AlibabaCloudSdkGoERROR)
		}
	}
	return nil
}

func (s *AlidnsService) DescribeAlidnsDomain(id string) (object alidns.DescribeDomainInfoResponse, err error) {
	request := alidns.CreateDescribeDomainInfoRequest()
	request.RegionId = s.client.RegionId

	request.DomainName = id

	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainInfo(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidDomainName.NoExist"}) {
			err = WrapErrorf(Error(GetNotFoundMessage("AlidnsDomain", id)), NotFoundMsg, ProviderERROR)
			return
		}
		err = WrapErrorf(err, DefaultErrorMsg, id, request.GetActionName(), AlibabaCloudSdkGoERROR)
		return
	}
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)
	response, _ := raw.(*alidns.DescribeDomainInfoResponse)
	return *response, nil
}

func (s *AlidnsService) DescribeAlidnsInstance(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeDnsProductInstance"
	request := map[string]interface{}{
		"RegionId":   s.client.RegionId,
		"InstanceId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 5*time.Second)
	err = resource.Retry(11*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidDnsProduct"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns:Instance", id)), NotFoundMsg, ProviderERROR, fmt.Sprint(response["RequestId"]))
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeAlidnsCustomLine(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeCustomLine"
	request := map[string]interface{}{
		"LineId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"DnsCustomLine.NotExists"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns::CustomLine", id)), NotFoundMsg, ProviderERROR)
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeCustomLine(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeCustomLine"
	request := map[string]interface{}{
		"LineId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"DnsCustomLine.NotExists"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns::CustomLine", id)), NotFoundMsg, ProviderERROR)
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeAlidnsGtmInstance(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeDnsGtmInstance"
	request := map[string]interface{}{
		"InstanceId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"DnsGtmInstance.NotExists"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns:GtmInstance", id)), NotFoundWithResponse, response)
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeAlidnsAddressPool(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeDnsGtmInstanceAddressPool"
	request := map[string]interface{}{
		"AddrPoolId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"DnsGtmAddrPool.NotExists"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns::AddressPool", id)), NotFoundMsg, ProviderERROR, fmt.Sprint(response["RequestId"]))
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeAlidnsAccessStrategy(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeDnsGtmAccessStrategy"
	request := map[string]interface{}{
		"StrategyId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		if IsExpectedErrors(err, []string{"DnsGtmAccessStrategy.NotExists"}) {
			return object, WrapErrorf(Error(GetNotFoundMessage("Alidns:DnsGtmAccessStrategy", id)), NotFoundWithResponse, response)
		}
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeAlidnsMonitorConfig(id string) (object map[string]interface{}, err error) {
	var response map[string]interface{}
	conn, err := s.client.NewAlidnsClient()
	if err != nil {
		return nil, WrapError(err)
	}
	action := "DescribeDnsGtmMonitorConfig"
	request := map[string]interface{}{
		"MonitorConfigId": id,
	}
	runtime := util.RuntimeOptions{}
	runtime.SetAutoretry(true)
	wait := incrementalWait(3*time.Second, 3*time.Second)
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		response, err = conn.DoRequest(StringPointer(action), nil, StringPointer("POST"), StringPointer("2015-01-09"), StringPointer("AK"), nil, request, &runtime)
		if err != nil {
			if NeedRetry(err) {
				wait()
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	addDebug(action, response, request)
	if err != nil {
		return object, WrapErrorf(err, DefaultErrorMsg, id, action, AlibabaCloudSdkGoERROR)
	}
	v, err := jsonpath.Get("$", response)
	if err != nil {
		return object, WrapErrorf(err, FailedGetAttributeMsg, id, "$", response)
	}
	object = v.(map[string]interface{})
	return object, nil
}

func (s *AlidnsService) DescribeDomainRecords(domainName string) ([]alidns.Record, error) {
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName

	// Call the DescribeDomainRecords API
	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecords(request)
	})
	if err != nil {
		err = WrapErrorf(err, "failed to describe domain records for domain %s: %w", domainName, err)
		return nil, err
	}

	// Debug log for raw response
	addDebug(request.GetActionName(), raw, request.RpcRequest, request)

	// Parse the response
	response, ok := raw.(*alidns.DescribeDomainRecordsResponse)
	if !ok {
		return nil, fmt.Errorf("failed to cast response to DescribeDomainRecordsResponse")
	}

	// Return the list of records
	return response.DomainRecords.Record, nil
}

func (s *AlidnsService) GetRecordByAttributes(domainName, rr, recordType, value string) (*alidns.Record, error) {
	records, err := s.DescribeDomainRecords(domainName)
	if err != nil {
		return nil, WrapError(err)
	}

	for _, record := range records {
		if record.RR == rr && record.Type == recordType && record.Value == value {
			return &record, nil
		}
	}

	return nil, nil // No matching record found
}

func (s *AlidnsService) CreateRecord(domainName, rr, recordType, value string, ttl int, line string) (string, error) {
	// Step 1: Check if the record already exists
	existingRecords, err := s.DescribeDomainRecords(domainName)
	if err != nil {
		return "", fmt.Errorf("failed to describe domain records: %s", err)
	}

	for _, record := range existingRecords {
		if record.RR == rr && record.Type == recordType && record.Value == value {
			log.Printf("[DEBUG] Record already exists: %s", record.RecordId)
			return record.RecordId, nil // Return existing recordId
		}
	}

	// Step 2: Create a new record
	request := alidns.CreateAddDomainRecordRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName
	request.RR = rr
	request.Type = recordType
	request.Value = value
	request.TTL = requests.NewInteger(ttl)
	request.Line = line

	response, err := s.client.WithAlidnsClient(func(client *alidns.Client) (interface{}, error) {
		return client.AddDomainRecord(request)
	})

	if err != nil {
		return "", fmt.Errorf("failed to create DNS record: %s", err)
	}

	resp, ok := response.(*alidns.AddDomainRecordResponse)
	if !ok {
		return "", fmt.Errorf("unexpected response type")
	}

	return resp.RecordId, nil
}

// UpdateRecord updates the attributes of an existing DNS record.
func (s *AlidnsService) UpdateRecord(recordID, domainName, rr, recordType, value, line string, ttl int) error {
	// Fetch the current state of the record
	existingRecord, err := s.DescribeDomainRecordById(recordID, domainName)
	if err != nil {
		return fmt.Errorf("failed to fetch existing record %s: %w", recordID, err)
	}

	// Convert ttl to int64 for comparison
	ttlInt64 := int64(ttl)

	// Compare the current state with the requested updates
	if existingRecord.RR == rr && existingRecord.Type == recordType &&
		existingRecord.Value == value && existingRecord.TTL == ttlInt64 {
		log.Printf("[DEBUG] No changes detected for record %s, skipping update", recordID)
		return nil
	}

	// Proceed with the update if there are changes
	request := alidns.CreateUpdateDomainRecordRequest()
	request.RegionId = s.client.RegionId
	request.RecordId = recordID
	request.RR = rr
	request.Type = recordType
	request.Value = value
	request.TTL = requests.NewInteger(ttl)
	request.Line = line

	_, err = s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.UpdateDomainRecord(request)
	})
	if err != nil {
		return WrapErrorf(err, "failed to update DNS record with ID %s", recordID)
	}

	return nil
}

// DeleteRecord deletes a DNS record by its ID.
func (s *AlidnsService) DeleteRecord(recordID string) error {
	request := alidns.CreateDeleteDomainRecordRequest()
	request.RegionId = s.client.RegionId
	request.RecordId = recordID

	_, err := s.client.WithAlidnsClient(func(client *alidns.Client) (interface{}, error) {
		return client.DeleteDomainRecord(request)
	})
	if err != nil {
		if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
			// Record already deleted or not found, safe to ignore
			return nil
		}
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	return nil
}

// SetRecordRemark sets a remark for a DNS record.
// Note: If Alibaba Cloud SDK does not support this operation, it should be adjusted accordingly.
func (s *AlidnsService) SetRecordRemark(recordID, remark string) error {
	// Check if Alibaba SDK supports UpdateDomainRemarkRequest and confirm its required fields.
	request := alidns.CreateUpdateDomainRemarkRequest()
	request.RegionId = s.client.RegionId
	request.Remark = remark

	// Workaround if RecordId is not supported:
	// Adjust the request parameters to use DomainName and RR instead of RecordId.
	// Example below uses DomainName and RR if required.

	// Add additional fields to locate the record
	// request.DomainName = "<Domain Name>" // Replace with actual domain name if required.
	// request.RR = "<RR Value>"           // Replace with actual RR if needed.

	_, err := s.client.WithAlidnsClient(func(client *alidns.Client) (interface{}, error) {
		return client.UpdateDomainRemark(request)
	})
	if err != nil {
		return fmt.Errorf("failed to set remark for DNS record: %w", err)
	}
	return nil
}

func (s *AlidnsService) SetRecordWeight(recordID string, weight int) error {
	request := alidns.CreateUpdateDNSSLBWeightRequest()
	request.RegionId = s.client.RegionId
	request.RecordId = recordID
	request.Weight = requests.NewInteger(weight)

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.UpdateDNSSLBWeight(request)
	})
	if err != nil {
		return WrapError(fmt.Errorf("failed to update weight for record %s: %w", recordID, err))
	}
	return nil
}

func isValidDomainName(domain string) bool {
	// Regex for validating domain names
	re := regexp.MustCompile(`^([a-zA-Z0-9-]{1,63}\.){1,255}[a-zA-Z]{2,63}$`)
	return re.MatchString(domain)
}

// GetRecordWeight fetches the weight of a specific record by its ID.
func (s *AlidnsService) GetRecordWeight(domainName, recordID string) (int, error) {
	// Create the request for DescribeDomainRecords
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName

	// Fetch the domain records
	response, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecords(request)
	})
	if err != nil {
		return 0, WrapError(err)
	}

	resp, ok := response.(*alidns.DescribeDomainRecordsResponse)
	if !ok {
		return 0, fmt.Errorf("failed to cast response to DescribeDomainRecordsResponse")
	}

	// Iterate through the records to find the matching record ID
	for _, record := range resp.DomainRecords.Record {
		if record.RecordId == recordID {
			return record.Weight, nil
		}
	}

	return 0, fmt.Errorf("record with ID %s not found", recordID)
}

// GetRecordRemark fetches the remark of a specific record by its ID.
func (s *AlidnsService) GetRecordRemark(recordID string) (string, error) {
	request := alidns.CreateDescribeDomainRecordInfoRequest()
	request.RecordId = recordID

	response, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecordInfo(request)
	})
	if err != nil {
		return "", WrapError(err)
	}

	resp, ok := response.(*alidns.DescribeDomainRecordInfoResponse)
	if !ok {
		return "", fmt.Errorf("failed to cast response to DescribeDomainRecordInfoResponse")
	}

	return resp.Remark, nil
}

// GetRecordStatus fetches the status of a specific record by its ID.
func (s *AlidnsService) GetRecordStatus(recordID string) (string, error) {
	request := alidns.CreateDescribeDomainRecordInfoRequest()
	request.RecordId = recordID

	response, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecordInfo(request)
	})
	if err != nil {
		return "", WrapError(err)
	}

	resp, ok := response.(*alidns.DescribeDomainRecordInfoResponse)
	if !ok {
		return "", fmt.Errorf("failed to cast response to DescribeDomainRecordInfoResponse")
	}

	return resp.Status, nil
}

// Helper function to find removed records
func findRemovedRecords(oldRecords, newRecords []interface{}) []interface{} {
	removed := []interface{}{}
	newRecordsMap := map[string]bool{}

	// Create a map of new records for comparison
	for _, record := range newRecords {
		r := record.(map[string]interface{})
		newRecordsMap[r["id"].(string)] = true
	}

	// Find records in old that are not in new
	for _, record := range oldRecords {
		r := record.(map[string]interface{})
		if !newRecordsMap[r["id"].(string)] {
			removed = append(removed, record)
		}
	}

	return removed
}

func (s *AlidnsService) DescribeDomainRecordById(recordID string, domainName string) (*alidns.Record, error) {
	// Create a request to fetch the domain records
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName

	// Fetch all records under the domain
	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecords(request)
	})
	if err != nil {
		return nil, WrapError(err)
	}

	// Cast the response to the expected type
	response, ok := raw.(*alidns.DescribeDomainRecordsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for DescribeDomainRecords: %T", raw)
	}

	// Search for the record with the specified record ID
	for _, record := range response.DomainRecords.Record {
		if record.RecordId == recordID {
			return &alidns.Record{
				RecordId:   record.RecordId,
				DomainName: record.DomainName,
				RR:         record.RR,
				Type:       record.Type,
				Value:      record.Value,
				TTL:        record.TTL,
				Line:       record.Line,
				Weight:     record.Weight,
				Remark:     record.Remark,
				Status:     record.Status,
			}, nil
		}
	}

	return nil, fmt.Errorf("record with ID %s not found in domain %s", recordID, domainName)
}

func (s *AlidnsService) EnableWRRStatus(domainName, rr string) error {
	request := alidns.CreateSetDNSSLBStatusRequest()
	request.DomainName = domainName
	request.SubDomain = fmt.Sprintf("%s.%s", rr, domainName)
	request.Open = requests.NewBoolean(true)

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.SetDNSSLBStatus(request)
	})
	return WrapError(err)
}

func (s *AlidnsService) DisableWRRStatus(domainName, rr string) error {
	request := alidns.CreateSetDNSSLBStatusRequest()
	request.DomainName = domainName
	request.SubDomain = fmt.Sprintf("%s.%s", rr, domainName)
	request.Open = requests.NewBoolean(false)

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.SetDNSSLBStatus(request)
	})
	return WrapError(err)
}

func (s *AlidnsService) GetWRRStatus(domainName, rr string) (string, error) {
	request := alidns.CreateDescribeDNSSLBSubDomainsRequest()
	request.DomainName = domainName

	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDNSSLBSubDomains(request)
	})
	if err != nil {
		return "", WrapError(err)
	}

	response, ok := raw.(*alidns.DescribeDNSSLBSubDomainsResponse)
	if !ok {
		return "", fmt.Errorf("unexpected response type for DescribeDNSSLBSubDomains: %T", raw)
	}

	for _, subDomain := range response.SlbSubDomains.SlbSubDomain {
		if subDomain.SubDomain == fmt.Sprintf("%s.%s", rr, domainName) {
			if subDomain.Open {
				return "ENABLE", nil
			}
			return "DISABLE", nil
		}
	}

	return "DISABLE", nil
}

// Helper function to check if a record is missing in the new state
func recordIsMissing(recordID string, oldRecords, newRecords []interface{}) bool {
	for _, newRecord := range newRecords {
		if newRecord.(map[string]interface{})["value"] == getValueByRecordID(recordID, oldRecords) {
			return false
		}
	}
	return true
}

// Helper function to get the value of a record by its ID
func getValueByRecordID(recordID string, records []interface{}) string {
	for _, record := range records {
		r := record.(map[string]interface{})
		if r["id"] == recordID {
			return r["value"].(string)
		}
	}
	return ""
}

func (s *AlidnsService) UpdateRecordRemark(recordID, remark string) error {
	request := alidns.CreateUpdateDomainRecordRemarkRequest()
	request.RecordId = recordID
	request.Remark = remark
	request.RegionId = s.client.RegionId

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.UpdateDomainRecordRemark(request)
	})
	return WrapError(err)
}

func (s *AlidnsService) SetRecordStatus(recordID, status string) error {
	request := alidns.CreateSetDomainRecordStatusRequest()
	request.RecordId = recordID
	request.Status = status

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.SetDomainRecordStatus(request)
	})
	return WrapError(err)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
