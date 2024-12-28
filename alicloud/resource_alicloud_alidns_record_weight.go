package alicloud

import (
	"fmt"
	"log"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/aliyun/terraform-provider-alicloud/alicloud/connectivity"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceAlicloudAlidnsRecordWeight() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudAlidnsRecordWeightCreate,
		Read:   resourceAlicloudAlidnsRecordWeightRead,
		Update: resourceAlicloudAlidnsRecordWeightUpdate,
		Delete: resourceAlicloudAlidnsRecordWeightDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAlicloudAlidnsRecordWeightImport,
		},
		Schema: map[string]*schema.Schema{
			"domain_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"rr": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"A", "NS", "MX", "TXT", "CNAME", "SRV", "AAAA", "CAA"}, false),
			},
			"line": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"priority": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"wrr_status": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "ENABLE",
				ValidateFunc: validation.StringInSlice([]string{"ENABLE", "DISABLE"}, false),
			},
			"records": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 50, // Maximum number of records supported
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"status": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"ENABLE", "DISABLE"}, false),
						},
						"remark": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"weight": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAlicloudAlidnsRecordWeightCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	domainName := d.Get("domain_name").(string)
	rr := d.Get("rr").(string)
	recordType := d.Get("type").(string)
	records := d.Get("records").([]interface{})
	line := d.Get("line").(string)

	var recordIDs []string

	// Step 1: Create records without weights
	for _, record := range records {
		r := record.(map[string]interface{})
		value := r["value"].(string)
		ttl := r["ttl"].(int)

		recordID, err := alidnsService.CreateRecord(domainName, rr, recordType, value, ttl, line)
		if err != nil {
			return fmt.Errorf("failed to create record: %s", err)
		}
		recordIDs = append(recordIDs, recordID)
	}

	// Step 2: Enable WRR status
	if d.Get("wrr_status").(string) == "ENABLE" {
		if err := alidnsService.EnableWRRStatus(domainName, rr); err != nil {
			return fmt.Errorf("failed to enable WRR: %s", err)
		}
	}

	// Step 3: Set weights for the records
	for i, record := range records {
		r := record.(map[string]interface{})
		weight := r["weight"].(int)

		if weight > 0 {
			err := alidnsService.SetRecordWeight(recordIDs[i], weight)
			if err != nil {
				return fmt.Errorf("failed to set weight for record %d: %s", i+1, err)
			}
		}
	}

	d.SetId(strings.Join(recordIDs, ","))
	return resourceAlicloudAlidnsRecordWeightRead(d, meta)
}

func resourceAlicloudAlidnsRecordWeightRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	// Retrieve the domain name from the schema
	domainName := d.Get("domain_name").(string)
	recordIDs := strings.Split(d.Id(), ",")
	validRecordIDs := []string{}
	validRecords := []map[string]interface{}{}

	for _, recordID := range recordIDs {
		// Attempt to fetch record details
		record, err := alidnsService.DescribeDomainRecordById(recordID, domainName)
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
				log.Printf("[DEBUG] Record with ID %s not found; skipping.", recordID)
				continue
			}
			return fmt.Errorf("failed to fetch record %s: %w", recordID, err)
		}

		log.Printf("[DEBUG] Fetched record: %+v", record)

		// Process valid records
		validRecordIDs = append(validRecordIDs, recordID)

		// Map the record details to the resource schema
		validRecords = append(validRecords, map[string]interface{}{
			"value":  record.Value,
			"ttl":    record.TTL,
			"weight": record.Weight,
			"remark": record.Remark,
			"status": record.Status,
		})
	}

	// Update the state to only include valid record IDs
	if len(validRecordIDs) == 0 {
		log.Printf("[DEBUG] No valid records found; removing resource from state.")
		d.SetId("")
		return nil
	}

	// Update the Terraform state
	d.Set("records", validRecords)
	d.SetId(strings.Join(validRecordIDs, ","))
	return nil
}

func resourceAlicloudAlidnsRecordWeightUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	domainName := d.Get("domain_name").(string)
	rr := d.Get("rr").(string)
	recordType := d.Get("type").(string)
	line := d.Get("line").(string)

	if d.HasChange("records") {
		_, new := d.GetChange("records")
		newRecords := new.([]interface{})
		recordIDs := strings.Split(d.Id(), ",")

		// Handle deletion of records no longer in the configuration
		if len(recordIDs) > len(newRecords) {
			for i := len(newRecords); i < len(recordIDs); i++ {
				recordID := recordIDs[i]
				err := alidnsService.DeleteRecord(recordID)
				if err != nil {
					return fmt.Errorf("failed to delete record %s: %w", recordID, err)
				}
			}
			recordIDs = recordIDs[:len(newRecords)]
		}

		if len(recordIDs) != len(newRecords) {
			return fmt.Errorf("mismatch between record IDs (%d) and new records (%d)", len(recordIDs), len(newRecords))
		}

		for i, record := range newRecords {
			r := record.(map[string]interface{})
			recordID := recordIDs[i]

			value := r["value"].(string)
			ttl := r["ttl"].(int)
			weight := r["weight"].(int)
			remark := r["remark"].(string)
			status := r["status"].(string)

			// Update record value, TTL, and type
			err := alidnsService.UpdateRecord(recordID, domainName, rr, recordType, value, line, ttl)
			if err != nil {
				return fmt.Errorf("failed to update record %s: %w", recordID, err)
			}

			// Update weight
			err = alidnsService.SetRecordWeight(recordID, weight)
			if err != nil {
				return fmt.Errorf("failed to update weight for record %s: %w", recordID, err)
			}

			// Update remark
			if remark != "" {
				err = alidnsService.UpdateRecordRemark(recordID, remark)
				if err != nil {
					return fmt.Errorf("failed to update remark for record %s: %w", recordID, err)
				}
			}

			// Update status
			if status != "" {
				err = alidnsService.SetRecordStatus(recordID, status)
				if err != nil {
					return fmt.Errorf("failed to update status for record %s: %w", recordID, err)
				}
			}
		}
	}

	return resourceAlicloudAlidnsRecordWeightRead(d, meta)
}

func resourceAlicloudAlidnsRecordWeightDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	recordIDs := strings.Split(d.Id(), ",")
	for _, recordID := range recordIDs {
		err := alidnsService.DeleteRecord(recordID)
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
				continue
			}
			return WrapError(fmt.Errorf("failed to delete record %s: %w", recordID, err))
		}
	}

	d.SetId("")
	return nil
}

func resourceAlicloudAlidnsRecordWeightImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	importID := d.Id()
	parts := strings.Split(importID, "/")
	var domainName, recordID string
	if len(parts) == 2 {
		domainName = parts[0]
		recordID = parts[1]
	} else {
		return nil, fmt.Errorf("import ID must be in the format domain_name/record_id")
	}

	record, err := alidnsService.DescribeDomainRecordById(recordID, domainName)
	if err != nil {
		return nil, WrapError(err)
	}

	d.SetId(record.RecordId)
	d.Set("domain_name", domainName)
	d.Set("rr", record.RR)
	d.Set("type", record.Type)
	d.Set("value", record.Value)
	d.Set("ttl", record.TTL)
	d.Set("line", record.Line)
	d.Set("weight", record.Weight)
	d.Set("remark", record.Remark)
	d.Set("status", record.Status)

	return []*schema.ResourceData{d}, nil
}

func (s *AlidnsService) SetWRRStatus(domainName, rr, status string) error {
	request := alidns.CreateSetDNSSLBStatusRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName
	request.SubDomain = fmt.Sprintf("%s.%s", rr, domainName)
	request.Open = requests.NewBoolean(status == "ENABLE")

	_, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.SetDNSSLBStatus(request)
	})
	return WrapError(err)
}

func (s *AlidnsService) DescribeDomainRecordById(recordID string, domainName string) (*alidns.Record, error) {
	// Request to fetch all records under the domain
	request := alidns.CreateDescribeDomainRecordsRequest()
	request.RegionId = s.client.RegionId
	request.DomainName = domainName

	// Make the API call to fetch the domain records
	raw, err := s.client.WithAlidnsClient(func(alidnsClient *alidns.Client) (interface{}, error) {
		return alidnsClient.DescribeDomainRecords(request)
	})
	if err != nil {
		return nil, WrapError(err)
	}

	// Cast the response
	response, ok := raw.(*alidns.DescribeDomainRecordsResponse)
	if !ok {
		return nil, fmt.Errorf("failed to cast response to DescribeDomainRecordsResponse")
	}

	// Search for the record with the given recordID
	for _, record := range response.DomainRecords.Record {
		if record.RecordId == recordID {
			// Return the matching record
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

	// If no record is found, return an error
	return nil, fmt.Errorf("record with ID %s not found in domain %s", recordID, domainName)
}
