package consumption

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"encoding/json"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/go-autorest/tracing"
	"github.com/shopspring/decimal"
	"net/http"
)

// The package's fully qualified name.
const fqdn = "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2017-11-30/consumption"

// Datagrain enumerates the values for datagrain.
type Datagrain string

const (
	// DailyGrain Daily grain of data
	DailyGrain Datagrain = "daily"
	// MonthlyGrain Monthly grain of data
	MonthlyGrain Datagrain = "monthly"
)

// PossibleDatagrainValues returns an array of possible values for the Datagrain const type.
func PossibleDatagrainValues() []Datagrain {
	return []Datagrain{DailyGrain, MonthlyGrain}
}

// ErrorDetails the details of the error.
type ErrorDetails struct {
	// Code - READ-ONLY; Error code.
	Code *string `json:"code,omitempty"`
	// Message - READ-ONLY; Error message indicating why the operation failed.
	Message *string `json:"message,omitempty"`
}

// ErrorResponse error response indicates that the service is not able to process the incoming request. The
// reason is provided in the error message.
type ErrorResponse struct {
	// Error - The details of the error.
	Error *ErrorDetails `json:"error,omitempty"`
}

// MeterDetails the properties of the meter detail.
type MeterDetails struct {
	// MeterName - READ-ONLY; The name of the meter, within the given meter category
	MeterName *string `json:"meterName,omitempty"`
	// MeterCategory - READ-ONLY; The category of the meter, for example, 'Cloud services', 'Networking', etc..
	MeterCategory *string `json:"meterCategory,omitempty"`
	// MeterSubCategory - READ-ONLY; The subcategory of the meter, for example, 'A6 Cloud services', 'ExpressRoute (IXP)', etc..
	MeterSubCategory *string `json:"meterSubCategory,omitempty"`
	// Unit - READ-ONLY; The unit in which the meter consumption is charged, for example, 'Hours', 'GB', etc.
	Unit *string `json:"unit,omitempty"`
	// MeterLocation - READ-ONLY; The location in which the Azure service is available.
	MeterLocation *string `json:"meterLocation,omitempty"`
	// TotalIncludedQuantity - READ-ONLY; The total included quantity associated with the offer.
	TotalIncludedQuantity *decimal.Decimal `json:"totalIncludedQuantity,omitempty"`
	// PretaxStandardRate - READ-ONLY; The pretax listing price.
	PretaxStandardRate *decimal.Decimal `json:"pretaxStandardRate,omitempty"`
}

// Operation a Consumption REST API operation.
type Operation struct {
	// Name - READ-ONLY; Operation name: {provider}/{resource}/{operation}.
	Name *string `json:"name,omitempty"`
	// Display - The object that represents the operation.
	Display *OperationDisplay `json:"display,omitempty"`
}

// OperationDisplay the object that represents the operation.
type OperationDisplay struct {
	// Provider - READ-ONLY; Service provider: Microsoft.Consumption.
	Provider *string `json:"provider,omitempty"`
	// Resource - READ-ONLY; Resource on which the operation is performed: UsageDetail, etc.
	Resource *string `json:"resource,omitempty"`
	// Operation - READ-ONLY; Operation type: Read, write, delete, etc.
	Operation *string `json:"operation,omitempty"`
}

// OperationListResult result of listing consumption operations. It contains a list of operations and a URL
// link to get the next set of results.
type OperationListResult struct {
	autorest.Response `json:"-"`
	// Value - READ-ONLY; List of consumption operations supported by the Microsoft.Consumption resource provider.
	Value *[]Operation `json:"value,omitempty"`
	// NextLink - READ-ONLY; URL to get the next set of operation list results if there are any.
	NextLink *string `json:"nextLink,omitempty"`
}

// OperationListResultIterator provides access to a complete listing of Operation values.
type OperationListResultIterator struct {
	i    int
	page OperationListResultPage
}

// NextWithContext advances to the next value.  If there was an error making
// the request the iterator does not advance and the error is returned.
func (iter *OperationListResultIterator) NextWithContext(ctx context.Context) (err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/OperationListResultIterator.NextWithContext")
		defer func() {
			sc := -1
			if iter.Response().Response.Response != nil {
				sc = iter.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	iter.i++
	if iter.i < len(iter.page.Values()) {
		return nil
	}
	err = iter.page.NextWithContext(ctx)
	if err != nil {
		iter.i--
		return err
	}
	iter.i = 0
	return nil
}

// Next advances to the next value.  If there was an error making
// the request the iterator does not advance and the error is returned.
// Deprecated: Use NextWithContext() instead.
func (iter *OperationListResultIterator) Next() error {
	return iter.NextWithContext(context.Background())
}

// NotDone returns true if the enumeration should be started or is not yet complete.
func (iter OperationListResultIterator) NotDone() bool {
	return iter.page.NotDone() && iter.i < len(iter.page.Values())
}

// Response returns the raw server response from the last page request.
func (iter OperationListResultIterator) Response() OperationListResult {
	return iter.page.Response()
}

// Value returns the current value or a zero-initialized value if the
// iterator has advanced beyond the end of the collection.
func (iter OperationListResultIterator) Value() Operation {
	if !iter.page.NotDone() {
		return Operation{}
	}
	return iter.page.Values()[iter.i]
}

// Creates a new instance of the OperationListResultIterator type.
func NewOperationListResultIterator(page OperationListResultPage) OperationListResultIterator {
	return OperationListResultIterator{page: page}
}

// IsEmpty returns true if the ListResult contains no values.
func (olr OperationListResult) IsEmpty() bool {
	return olr.Value == nil || len(*olr.Value) == 0
}

// operationListResultPreparer prepares a request to retrieve the next set of results.
// It returns nil if no more results exist.
func (olr OperationListResult) operationListResultPreparer(ctx context.Context) (*http.Request, error) {
	if olr.NextLink == nil || len(to.String(olr.NextLink)) < 1 {
		return nil, nil
	}
	return autorest.Prepare((&http.Request{}).WithContext(ctx),
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(olr.NextLink)))
}

// OperationListResultPage contains a page of Operation values.
type OperationListResultPage struct {
	fn  func(context.Context, OperationListResult) (OperationListResult, error)
	olr OperationListResult
}

// NextWithContext advances to the next page of values.  If there was an error making
// the request the page does not advance and the error is returned.
func (page *OperationListResultPage) NextWithContext(ctx context.Context) (err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/OperationListResultPage.NextWithContext")
		defer func() {
			sc := -1
			if page.Response().Response.Response != nil {
				sc = page.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	next, err := page.fn(ctx, page.olr)
	if err != nil {
		return err
	}
	page.olr = next
	return nil
}

// Next advances to the next page of values.  If there was an error making
// the request the page does not advance and the error is returned.
// Deprecated: Use NextWithContext() instead.
func (page *OperationListResultPage) Next() error {
	return page.NextWithContext(context.Background())
}

// NotDone returns true if the page enumeration should be started or is not yet complete.
func (page OperationListResultPage) NotDone() bool {
	return !page.olr.IsEmpty()
}

// Response returns the raw server response from the last page request.
func (page OperationListResultPage) Response() OperationListResult {
	return page.olr
}

// Values returns the slice of values for the current page or nil if there are no values.
func (page OperationListResultPage) Values() []Operation {
	if page.olr.IsEmpty() {
		return nil
	}
	return *page.olr.Value
}

// Creates a new instance of the OperationListResultPage type.
func NewOperationListResultPage(getNextPage func(context.Context, OperationListResult) (OperationListResult, error)) OperationListResultPage {
	return OperationListResultPage{fn: getNextPage}
}

// ReservationDetails reservation details resource.
type ReservationDetails struct {
	*ReservationDetailsProperties `json:"properties,omitempty"`
	// ID - READ-ONLY; Resource Id.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Tags - READ-ONLY; Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for ReservationDetails.
func (rd ReservationDetails) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if rd.ReservationDetailsProperties != nil {
		objectMap["properties"] = rd.ReservationDetailsProperties
	}
	return json.Marshal(objectMap)
}

// UnmarshalJSON is the custom unmarshaler for ReservationDetails struct.
func (rd *ReservationDetails) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case "properties":
			if v != nil {
				var reservationDetailsProperties ReservationDetailsProperties
				err = json.Unmarshal(*v, &reservationDetailsProperties)
				if err != nil {
					return err
				}
				rd.ReservationDetailsProperties = &reservationDetailsProperties
			}
		case "id":
			if v != nil {
				var ID string
				err = json.Unmarshal(*v, &ID)
				if err != nil {
					return err
				}
				rd.ID = &ID
			}
		case "name":
			if v != nil {
				var name string
				err = json.Unmarshal(*v, &name)
				if err != nil {
					return err
				}
				rd.Name = &name
			}
		case "type":
			if v != nil {
				var typeVar string
				err = json.Unmarshal(*v, &typeVar)
				if err != nil {
					return err
				}
				rd.Type = &typeVar
			}
		case "tags":
			if v != nil {
				var tags map[string]*string
				err = json.Unmarshal(*v, &tags)
				if err != nil {
					return err
				}
				rd.Tags = tags
			}
		}
	}

	return nil
}

// ReservationDetailsListResult result of listing reservation details.
type ReservationDetailsListResult struct {
	autorest.Response `json:"-"`
	// Value - READ-ONLY; The list of reservation details.
	Value *[]ReservationDetails `json:"value,omitempty"`
}

// ReservationDetailsProperties the properties of the reservation details.
type ReservationDetailsProperties struct {
	// ReservationOrderID - READ-ONLY; The reservation order ID is the identifier for a reservation purchase. Each reservation order ID represents a single purchase transaction. A reservation order contains reservations. The reservation order specifies the VM size and region for the reservations.
	ReservationOrderID *string `json:"reservationOrderId,omitempty"`
	// ReservationID - READ-ONLY; The reservation ID is the identifier of a reservation within a reservation order. Each reservation is the grouping for applying the benefit scope and also specifies the number of instances to which the reservation benefit can be applied to.
	ReservationID *string `json:"reservationId,omitempty"`
	// SkuName - READ-ONLY; This is the ARM Sku name. It can be used to join with the serviceType field in additional info in usage records.
	SkuName *string `json:"skuName,omitempty"`
	// ReservedHours - READ-ONLY; This is the total hours reserved for the day. E.g. if reservation for 1 instance was made on 1 PM, this will be 11 hours for that day and 24 hours from subsequent days.
	ReservedHours *decimal.Decimal `json:"reservedHours,omitempty"`
	// UsageDate - READ-ONLY; The date on which consumption occurred.
	UsageDate *date.Time `json:"usageDate,omitempty"`
	// UsedHours - READ-ONLY; This is the total hours used by the instance.
	UsedHours *decimal.Decimal `json:"usedHours,omitempty"`
	// InstanceID - READ-ONLY; This identifier is the name of the resource or the fully qualified Resource ID.
	InstanceID *string `json:"instanceId,omitempty"`
	// TotalReservedQuantity - READ-ONLY; This is the total count of instances that are reserved for the reservationId.
	TotalReservedQuantity *decimal.Decimal `json:"totalReservedQuantity,omitempty"`
}

// ReservationSummaries reservation summaries resource.
type ReservationSummaries struct {
	*ReservationSummariesProperties `json:"properties,omitempty"`
	// ID - READ-ONLY; Resource Id.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Tags - READ-ONLY; Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for ReservationSummaries.
func (rs ReservationSummaries) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if rs.ReservationSummariesProperties != nil {
		objectMap["properties"] = rs.ReservationSummariesProperties
	}
	return json.Marshal(objectMap)
}

// UnmarshalJSON is the custom unmarshaler for ReservationSummaries struct.
func (rs *ReservationSummaries) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case "properties":
			if v != nil {
				var reservationSummariesProperties ReservationSummariesProperties
				err = json.Unmarshal(*v, &reservationSummariesProperties)
				if err != nil {
					return err
				}
				rs.ReservationSummariesProperties = &reservationSummariesProperties
			}
		case "id":
			if v != nil {
				var ID string
				err = json.Unmarshal(*v, &ID)
				if err != nil {
					return err
				}
				rs.ID = &ID
			}
		case "name":
			if v != nil {
				var name string
				err = json.Unmarshal(*v, &name)
				if err != nil {
					return err
				}
				rs.Name = &name
			}
		case "type":
			if v != nil {
				var typeVar string
				err = json.Unmarshal(*v, &typeVar)
				if err != nil {
					return err
				}
				rs.Type = &typeVar
			}
		case "tags":
			if v != nil {
				var tags map[string]*string
				err = json.Unmarshal(*v, &tags)
				if err != nil {
					return err
				}
				rs.Tags = tags
			}
		}
	}

	return nil
}

// ReservationSummariesListResult result of listing reservation summaries.
type ReservationSummariesListResult struct {
	autorest.Response `json:"-"`
	// Value - READ-ONLY; The list of reservation summaries.
	Value *[]ReservationSummaries `json:"value,omitempty"`
}

// ReservationSummariesProperties the properties of the reservation summaries.
type ReservationSummariesProperties struct {
	// ReservationOrderID - READ-ONLY; The reservation order ID is the identifier for a reservation purchase. Each reservation order ID represents a single purchase transaction. A reservation order contains reservations. The reservation order specifies the VM size and region for the reservations.
	ReservationOrderID *string `json:"reservationOrderId,omitempty"`
	// ReservationID - READ-ONLY; The reservation ID is the identifier of a reservation within a reservation order. Each reservation is the grouping for applying the benefit scope and also specifies the number of instances to which the reservation benefit can be applied to.
	ReservationID *string `json:"reservationId,omitempty"`
	// SkuName - READ-ONLY; This is the ARM Sku name. It can be used to join with the serviceType field in additional info in usage records.
	SkuName *string `json:"skuName,omitempty"`
	// ReservedHours - READ-ONLY; This is the total hours reserved. E.g. if reservation for 1 instance was made on 1 PM, this will be 11 hours for that day and 24 hours from subsequent days
	ReservedHours *decimal.Decimal `json:"reservedHours,omitempty"`
	// UsageDate - READ-ONLY; Data corresponding to the utilization record. If the grain of data is monthly, it will be first day of month.
	UsageDate *date.Time `json:"usageDate,omitempty"`
	// UsedHours - READ-ONLY; Total used hours by the reservation
	UsedHours *decimal.Decimal `json:"usedHours,omitempty"`
	// MinUtilizationPercentage - READ-ONLY; This is the minimum hourly utilization in the usage time (day or month). E.g. if usage record corresponds to 12/10/2017 and on that for hour 4 and 5, utilization was 10%, this field will return 10% for that day
	MinUtilizationPercentage *decimal.Decimal `json:"minUtilizationPercentage,omitempty"`
	// AvgUtilizationPercentage - READ-ONLY; This is average utilization for the entire time range. (day or month depending on the grain)
	AvgUtilizationPercentage *decimal.Decimal `json:"avgUtilizationPercentage,omitempty"`
	// MaxUtilizationPercentage - READ-ONLY; This is the maximum hourly utilization in the usage time (day or month). E.g. if usage record corresponds to 12/10/2017 and on that for hour 4 and 5, utilization was 100%, this field will return 100% for that day.
	MaxUtilizationPercentage *decimal.Decimal `json:"maxUtilizationPercentage,omitempty"`
}

// Resource the Resource model definition.
type Resource struct {
	// ID - READ-ONLY; Resource Id.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Tags - READ-ONLY; Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for Resource.
func (r Resource) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	return json.Marshal(objectMap)
}

// UsageDetail an usage detail resource.
type UsageDetail struct {
	*UsageDetailProperties `json:"properties,omitempty"`
	// ID - READ-ONLY; Resource Id.
	ID *string `json:"id,omitempty"`
	// Name - READ-ONLY; Resource name.
	Name *string `json:"name,omitempty"`
	// Type - READ-ONLY; Resource type.
	Type *string `json:"type,omitempty"`
	// Tags - READ-ONLY; Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for UsageDetail.
func (ud UsageDetail) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if ud.UsageDetailProperties != nil {
		objectMap["properties"] = ud.UsageDetailProperties
	}
	return json.Marshal(objectMap)
}

// UnmarshalJSON is the custom unmarshaler for UsageDetail struct.
func (ud *UsageDetail) UnmarshalJSON(body []byte) error {
	var m map[string]*json.RawMessage
	err := json.Unmarshal(body, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case "properties":
			if v != nil {
				var usageDetailProperties UsageDetailProperties
				err = json.Unmarshal(*v, &usageDetailProperties)
				if err != nil {
					return err
				}
				ud.UsageDetailProperties = &usageDetailProperties
			}
		case "id":
			if v != nil {
				var ID string
				err = json.Unmarshal(*v, &ID)
				if err != nil {
					return err
				}
				ud.ID = &ID
			}
		case "name":
			if v != nil {
				var name string
				err = json.Unmarshal(*v, &name)
				if err != nil {
					return err
				}
				ud.Name = &name
			}
		case "type":
			if v != nil {
				var typeVar string
				err = json.Unmarshal(*v, &typeVar)
				if err != nil {
					return err
				}
				ud.Type = &typeVar
			}
		case "tags":
			if v != nil {
				var tags map[string]*string
				err = json.Unmarshal(*v, &tags)
				if err != nil {
					return err
				}
				ud.Tags = tags
			}
		}
	}

	return nil
}

// UsageDetailProperties the properties of the usage detail.
type UsageDetailProperties struct {
	// BillingPeriodID - READ-ONLY; The id of the billing period resource that the usage belongs to.
	BillingPeriodID *string `json:"billingPeriodId,omitempty"`
	// InvoiceID - READ-ONLY; The id of the invoice resource that the usage belongs to.
	InvoiceID *string `json:"invoiceId,omitempty"`
	// UsageStart - READ-ONLY; The start of the date time range covered by the usage detail.
	UsageStart *date.Time `json:"usageStart,omitempty"`
	// UsageEnd - READ-ONLY; The end of the date time range covered by the usage detail.
	UsageEnd *date.Time `json:"usageEnd,omitempty"`
	// InstanceName - READ-ONLY; The name of the resource instance that the usage is about.
	InstanceName *string `json:"instanceName,omitempty"`
	// InstanceID - READ-ONLY; The uri of the resource instance that the usage is about.
	InstanceID *string `json:"instanceId,omitempty"`
	// InstanceLocation - READ-ONLY; The location of the resource instance that the usage is about.
	InstanceLocation *string `json:"instanceLocation,omitempty"`
	// Currency - READ-ONLY; The ISO currency in which the meter is charged, for example, USD.
	Currency *string `json:"currency,omitempty"`
	// UsageQuantity - READ-ONLY; The quantity of usage.
	UsageQuantity *decimal.Decimal `json:"usageQuantity,omitempty"`
	// BillableQuantity - READ-ONLY; The billable usage quantity.
	BillableQuantity *decimal.Decimal `json:"billableQuantity,omitempty"`
	// PretaxCost - READ-ONLY; The amount of cost before tax.
	PretaxCost *decimal.Decimal `json:"pretaxCost,omitempty"`
	// IsEstimated - READ-ONLY; The estimated usage is subject to change.
	IsEstimated *bool `json:"isEstimated,omitempty"`
	// MeterID - READ-ONLY; The meter id.
	MeterID *string `json:"meterId,omitempty"`
	// MeterDetails - READ-ONLY; The details about the meter. By default this is not populated, unless it's specified in $expand.
	MeterDetails *MeterDetails `json:"meterDetails,omitempty"`
	// SubscriptionGUID - READ-ONLY; Subscription guid.
	SubscriptionGUID *string `json:"subscriptionGuid,omitempty"`
	// SubscriptionName - READ-ONLY; Subscription name.
	SubscriptionName *string `json:"subscriptionName,omitempty"`
	// AccountName - READ-ONLY; Account name.
	AccountName *string `json:"accountName,omitempty"`
	// DepartmentName - READ-ONLY; Department name.
	DepartmentName *string `json:"departmentName,omitempty"`
	// Product - READ-ONLY; Product name.
	Product *string `json:"product,omitempty"`
	// ConsumedService - READ-ONLY; Consumed service name.
	ConsumedService *string `json:"consumedService,omitempty"`
	// CostCenter - READ-ONLY; The cost center of this department if it is a department and a costcenter exists
	CostCenter *string `json:"costCenter,omitempty"`
	// AdditionalProperties - READ-ONLY; Additional details of this usage item. By default this is not populated, unless it's specified in $expand.
	AdditionalProperties *string `json:"additionalProperties,omitempty"`
}

// UsageDetailsListResult result of listing usage details. It contains a list of available usage details in
// reverse chronological order by billing period.
type UsageDetailsListResult struct {
	autorest.Response `json:"-"`
	// Value - READ-ONLY; The list of usage details.
	Value *[]UsageDetail `json:"value,omitempty"`
	// NextLink - READ-ONLY; The link (url) to the next page of results.
	NextLink *string `json:"nextLink,omitempty"`
}

// UsageDetailsListResultIterator provides access to a complete listing of UsageDetail values.
type UsageDetailsListResultIterator struct {
	i    int
	page UsageDetailsListResultPage
}

// NextWithContext advances to the next value.  If there was an error making
// the request the iterator does not advance and the error is returned.
func (iter *UsageDetailsListResultIterator) NextWithContext(ctx context.Context) (err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/UsageDetailsListResultIterator.NextWithContext")
		defer func() {
			sc := -1
			if iter.Response().Response.Response != nil {
				sc = iter.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	iter.i++
	if iter.i < len(iter.page.Values()) {
		return nil
	}
	err = iter.page.NextWithContext(ctx)
	if err != nil {
		iter.i--
		return err
	}
	iter.i = 0
	return nil
}

// Next advances to the next value.  If there was an error making
// the request the iterator does not advance and the error is returned.
// Deprecated: Use NextWithContext() instead.
func (iter *UsageDetailsListResultIterator) Next() error {
	return iter.NextWithContext(context.Background())
}

// NotDone returns true if the enumeration should be started or is not yet complete.
func (iter UsageDetailsListResultIterator) NotDone() bool {
	return iter.page.NotDone() && iter.i < len(iter.page.Values())
}

// Response returns the raw server response from the last page request.
func (iter UsageDetailsListResultIterator) Response() UsageDetailsListResult {
	return iter.page.Response()
}

// Value returns the current value or a zero-initialized value if the
// iterator has advanced beyond the end of the collection.
func (iter UsageDetailsListResultIterator) Value() UsageDetail {
	if !iter.page.NotDone() {
		return UsageDetail{}
	}
	return iter.page.Values()[iter.i]
}

// Creates a new instance of the UsageDetailsListResultIterator type.
func NewUsageDetailsListResultIterator(page UsageDetailsListResultPage) UsageDetailsListResultIterator {
	return UsageDetailsListResultIterator{page: page}
}

// IsEmpty returns true if the ListResult contains no values.
func (udlr UsageDetailsListResult) IsEmpty() bool {
	return udlr.Value == nil || len(*udlr.Value) == 0
}

// usageDetailsListResultPreparer prepares a request to retrieve the next set of results.
// It returns nil if no more results exist.
func (udlr UsageDetailsListResult) usageDetailsListResultPreparer(ctx context.Context) (*http.Request, error) {
	if udlr.NextLink == nil || len(to.String(udlr.NextLink)) < 1 {
		return nil, nil
	}
	return autorest.Prepare((&http.Request{}).WithContext(ctx),
		autorest.AsJSON(),
		autorest.AsGet(),
		autorest.WithBaseURL(to.String(udlr.NextLink)))
}

// UsageDetailsListResultPage contains a page of UsageDetail values.
type UsageDetailsListResultPage struct {
	fn   func(context.Context, UsageDetailsListResult) (UsageDetailsListResult, error)
	udlr UsageDetailsListResult
}

// NextWithContext advances to the next page of values.  If there was an error making
// the request the page does not advance and the error is returned.
func (page *UsageDetailsListResultPage) NextWithContext(ctx context.Context) (err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/UsageDetailsListResultPage.NextWithContext")
		defer func() {
			sc := -1
			if page.Response().Response.Response != nil {
				sc = page.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	next, err := page.fn(ctx, page.udlr)
	if err != nil {
		return err
	}
	page.udlr = next
	return nil
}

// Next advances to the next page of values.  If there was an error making
// the request the page does not advance and the error is returned.
// Deprecated: Use NextWithContext() instead.
func (page *UsageDetailsListResultPage) Next() error {
	return page.NextWithContext(context.Background())
}

// NotDone returns true if the page enumeration should be started or is not yet complete.
func (page UsageDetailsListResultPage) NotDone() bool {
	return !page.udlr.IsEmpty()
}

// Response returns the raw server response from the last page request.
func (page UsageDetailsListResultPage) Response() UsageDetailsListResult {
	return page.udlr
}

// Values returns the slice of values for the current page or nil if there are no values.
func (page UsageDetailsListResultPage) Values() []UsageDetail {
	if page.udlr.IsEmpty() {
		return nil
	}
	return *page.udlr.Value
}

// Creates a new instance of the UsageDetailsListResultPage type.
func NewUsageDetailsListResultPage(getNextPage func(context.Context, UsageDetailsListResult) (UsageDetailsListResult, error)) UsageDetailsListResultPage {
	return UsageDetailsListResultPage{fn: getNextPage}
}
