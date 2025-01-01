package alicloud

import (
	"fmt"
	"log"
	"strings"

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
	line := d.Get("line").(string)
	records := d.Get("records").([]interface{})

	var recordIDs []string

	// Create records for each entry in the records list
	for _, record := range records {
		r := record.(map[string]interface{})
		value := r["value"].(string)
		ttl := r["ttl"].(int)
		remark := r["remark"].(string)

		// Create the DNS record
		recordID, err := alidnsService.CreateRecord(domainName, rr, recordType, value, ttl, line)
		if err != nil {
			return fmt.Errorf("failed to create record: %s", err)
		}
		recordIDs = append(recordIDs, recordID)

		// Update the remark for the created record
		if remark != "" {
			if err := alidnsService.UpdateRecordRemark(recordID, remark); err != nil {
				return fmt.Errorf("failed to set remark for record %s: %s", recordID, err)
			}
		}
	}

	// Enable WRR status if required
	if d.Get("wrr_status").(string) == "ENABLE" {
		if err := alidnsService.EnableWRRStatus(domainName, rr); err != nil {
			return fmt.Errorf("failed to enable WRR: %s", err)
		}
	}

	// Set weights for each record
	for i, record := range records {
		r := record.(map[string]interface{})
		weight := r["weight"].(int)

		if err := alidnsService.SetRecordWeight(recordIDs[i], weight); err != nil {
			return fmt.Errorf("failed to set weight for record %d: %s", i+1, err)
		}
	}

	d.SetId(strings.Join(recordIDs, ","))
	return resourceAlicloudAlidnsRecordWeightRead(d, meta)
}

func resourceAlicloudAlidnsRecordWeightRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	domainName := d.Get("domain_name").(string)
	recordIDs := strings.Split(d.Id(), ",")

	validRecords := []map[string]interface{}{}
	validRecordIDs := []string{}

	for _, recordID := range recordIDs {
		record, err := alidnsService.DescribeDomainRecordById(recordID, domainName)
		if err != nil {
			if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
				log.Printf("[DEBUG] Record ID %s not found, skipping.", recordID)
				continue
			}
			return fmt.Errorf("failed to fetch existing record %s: %w", recordID, err)
		}

		validRecordIDs = append(validRecordIDs, recordID)
		validRecords = append(validRecords, map[string]interface{}{
			"value":  record.Value,
			"ttl":    record.TTL,
			"weight": record.Weight,
			"remark": record.Remark,
			"status": record.Status,
		})
	}

	if len(validRecordIDs) == 0 {
		log.Printf("[DEBUG] No valid records found; removing resource from state.")
		d.SetId("")
		return nil
	}

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
		recordIDs := strings.Split(d.Id(), ",")
		old, new := d.GetChange("records")
		oldRecords := old.([]interface{})
		newRecords := new.([]interface{})

		// Use oldRecords to find records that need deletion
		for _, oldRecord := range oldRecords {
			old := oldRecord.(map[string]interface{})
			oldValue := old["value"].(string)

			found := false
			for _, newRecord := range newRecords {
				new := newRecord.(map[string]interface{})
				if oldValue == new["value"].(string) {
					found = true
					break
				}
			}

			if !found {
				log.Printf("[DEBUG] Old record %s no longer exists in new records", oldValue)
				// Logic for deletion...
			}
		}

		updatedRecordIDs := []string{}

		// Step 1: Update or create records based on record ID
		for i, record := range newRecords {
			r := record.(map[string]interface{})
			value, ok := r["value"].(string)
			if !ok || value == "" {
				return fmt.Errorf("record value is missing or invalid")
			}

			ttl, ok := r["ttl"].(int)
			if !ok {
				return fmt.Errorf("record TTL is missing or invalid")
			}

			remark, _ := r["remark"].(string)
			status, _ := r["status"].(string)
			weight, _ := r["weight"].(int)

			if i < len(recordIDs) {
				// Update the existing record by ID
				recordID := recordIDs[i]
				log.Printf("[DEBUG] Updating existing record ID: %s", recordID)

				// Update basic attributes
				if err := alidnsService.UpdateRecord(recordID, domainName, rr, recordType, value, line, ttl); err != nil {
					return fmt.Errorf("failed to update record %s: %w", recordID, err)
				}

				// Update remark
				if remark != "" {
					if err := alidnsService.UpdateRecordRemark(recordID, remark); err != nil {
						return fmt.Errorf("failed to update remark for record %s: %w", recordID, err)
					}
				}

				// Update status
				if status != "" {
					if err := alidnsService.SetRecordStatus(recordID, status); err != nil {
						return fmt.Errorf("failed to update status for record %s: %w", recordID, err)
					}
				}

				// Update weight
				if weight > 0 {
					if err := alidnsService.SetRecordWeight(recordID, weight); err != nil {
						return fmt.Errorf("failed to set weight for record %s: %w", recordID, err)
					}
				}

				updatedRecordIDs = append(updatedRecordIDs, recordID)
			} else {
				// Create a new record if it doesn't exist
				log.Printf("[DEBUG] Creating new record with value: %s", value)
				recordID, err := alidnsService.CreateRecord(domainName, rr, recordType, value, ttl, line)
				if err != nil {
					return fmt.Errorf("failed to create record: %w", err)
				}
				updatedRecordIDs = append(updatedRecordIDs, recordID)
			}
		}

		// Step 2: Delete extra records not in newRecords
		for i := len(newRecords); i < len(recordIDs); i++ {
			recordID := recordIDs[i]
			log.Printf("[DEBUG] Deleting record ID: %s", recordID)
			if err := alidnsService.DeleteRecord(recordID); err != nil {
				if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
					log.Printf("[DEBUG] Record ID %s not found; skipping deletion.", recordID)
					continue
				}
				return fmt.Errorf("failed to delete record %s: %w", recordID, err)
			}
		}

		// Step 3: Update WRR status if changed
		if d.HasChange("wrr_status") {
			newStatus := d.Get("wrr_status").(string)
			if newStatus == "ENABLE" {
				if err := alidnsService.EnableWRRStatus(domainName, rr); err != nil {
					return fmt.Errorf("failed to enable WRR status: %w", err)
				}
			} else if newStatus == "DISABLE" {
				if err := alidnsService.DisableWRRStatus(domainName, rr); err != nil {
					return fmt.Errorf("failed to disable WRR status: %w", err)
				}
			}
		}

		// Step 4: Update state
		d.SetId(strings.Join(updatedRecordIDs, ","))
	}

	return resourceAlicloudAlidnsRecordWeightRead(d, meta)
}

func resourceAlicloudAlidnsRecordWeightDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*connectivity.AliyunClient)
	alidnsService := AlidnsService{client}

	recordIDs := strings.Split(d.Id(), ",")
	for _, recordID := range recordIDs {
		if err := alidnsService.DeleteRecord(recordID); err != nil {
			if IsExpectedErrors(err, []string{"InvalidRecordId.NotFound"}) {
				continue
			}
			return fmt.Errorf("failed to delete record %s: %w", recordID, err)
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
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid import ID format: expected domain_name/record_id")
	}

	domainName := parts[0]
	recordID := parts[1]

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
