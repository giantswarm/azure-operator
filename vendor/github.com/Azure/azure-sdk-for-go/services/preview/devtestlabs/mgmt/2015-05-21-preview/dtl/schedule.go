package dtl

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
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/tracing"
	"net/http"
)

// ScheduleClient is the azure DevTest Labs REST API version 2015-05-21-preview.
type ScheduleClient struct {
	BaseClient
}

// NewScheduleClient creates an instance of the ScheduleClient client.
func NewScheduleClient(subscriptionID string) ScheduleClient {
	return NewScheduleClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewScheduleClientWithBaseURI creates an instance of the ScheduleClient client.
func NewScheduleClientWithBaseURI(baseURI string, subscriptionID string) ScheduleClient {
	return ScheduleClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdateResource create or replace an existing schedule. This operation can take a while to complete.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// name - the name of the schedule.
func (client ScheduleClient) CreateOrUpdateResource(ctx context.Context, resourceGroupName string, labName string, name string, schedule Schedule) (result ScheduleCreateOrUpdateResourceFuture, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.CreateOrUpdateResource")
		defer func() {
			sc := -1
			if result.Response() != nil {
				sc = result.Response().StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.CreateOrUpdateResourcePreparer(ctx, resourceGroupName, labName, name, schedule)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "CreateOrUpdateResource", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateResourceSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "CreateOrUpdateResource", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdateResourcePreparer prepares the CreateOrUpdateResource request.
func (client ScheduleClient) CreateOrUpdateResourcePreparer(ctx context.Context, resourceGroupName string, labName string, name string, schedule Schedule) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"name":              autorest.Encode("path", name),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}", pathParameters),
		autorest.WithJSON(schedule),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateResourceSender sends the CreateOrUpdateResource request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) CreateOrUpdateResourceSender(req *http.Request) (future ScheduleCreateOrUpdateResourceFuture, err error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req, sd...)
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// CreateOrUpdateResourceResponder handles the response to the CreateOrUpdateResource request. The method always
// closes the http.Response Body.
func (client ScheduleClient) CreateOrUpdateResourceResponder(resp *http.Response) (result Schedule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// DeleteResource delete schedule. This operation can take a while to complete.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// name - the name of the schedule.
func (client ScheduleClient) DeleteResource(ctx context.Context, resourceGroupName string, labName string, name string) (result ScheduleDeleteResourceFuture, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.DeleteResource")
		defer func() {
			sc := -1
			if result.Response() != nil {
				sc = result.Response().StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.DeleteResourcePreparer(ctx, resourceGroupName, labName, name)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "DeleteResource", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteResourceSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "DeleteResource", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeleteResourcePreparer prepares the DeleteResource request.
func (client ScheduleClient) DeleteResourcePreparer(ctx context.Context, resourceGroupName string, labName string, name string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"name":              autorest.Encode("path", name),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteResourceSender sends the DeleteResource request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) DeleteResourceSender(req *http.Request) (future ScheduleDeleteResourceFuture, err error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req, sd...)
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// DeleteResourceResponder handles the response to the DeleteResource request. The method always
// closes the http.Response Body.
func (client ScheduleClient) DeleteResourceResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Execute execute a schedule. This operation can take a while to complete.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// name - the name of the schedule.
func (client ScheduleClient) Execute(ctx context.Context, resourceGroupName string, labName string, name string) (result ScheduleExecuteFuture, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.Execute")
		defer func() {
			sc := -1
			if result.Response() != nil {
				sc = result.Response().StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.ExecutePreparer(ctx, resourceGroupName, labName, name)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "Execute", nil, "Failure preparing request")
		return
	}

	result, err = client.ExecuteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "Execute", result.Response(), "Failure sending request")
		return
	}

	return
}

// ExecutePreparer prepares the Execute request.
func (client ScheduleClient) ExecutePreparer(ctx context.Context, resourceGroupName string, labName string, name string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"name":              autorest.Encode("path", name),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}/execute", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ExecuteSender sends the Execute request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) ExecuteSender(req *http.Request) (future ScheduleExecuteFuture, err error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req, sd...)
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// ExecuteResponder handles the response to the Execute request. The method always
// closes the http.Response Body.
func (client ScheduleClient) ExecuteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// GetResource get schedule.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// name - the name of the schedule.
func (client ScheduleClient) GetResource(ctx context.Context, resourceGroupName string, labName string, name string) (result Schedule, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.GetResource")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.GetResourcePreparer(ctx, resourceGroupName, labName, name)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "GetResource", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetResourceSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "GetResource", resp, "Failure sending request")
		return
	}

	result, err = client.GetResourceResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "GetResource", resp, "Failure responding to request")
	}

	return
}

// GetResourcePreparer prepares the GetResource request.
func (client ScheduleClient) GetResourcePreparer(ctx context.Context, resourceGroupName string, labName string, name string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"name":              autorest.Encode("path", name),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetResourceSender sends the GetResource request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) GetResourceSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	return autorest.SendWithSender(client, req, sd...)
}

// GetResourceResponder handles the response to the GetResource request. The method always
// closes the http.Response Body.
func (client ScheduleClient) GetResourceResponder(resp *http.Response) (result Schedule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// List list schedules.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// filter - the filter to apply on the operation.
func (client ScheduleClient) List(ctx context.Context, resourceGroupName string, labName string, filter string, top *int32, orderBy string) (result ResponseWithContinuationSchedulePage, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.List")
		defer func() {
			sc := -1
			if result.rwcs.Response.Response != nil {
				sc = result.rwcs.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	result.fn = client.listNextResults
	req, err := client.ListPreparer(ctx, resourceGroupName, labName, filter, top, orderBy)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "List", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.rwcs.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "List", resp, "Failure sending request")
		return
	}

	result.rwcs, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "List", resp, "Failure responding to request")
	}

	return
}

// ListPreparer prepares the List request.
func (client ScheduleClient) ListPreparer(ctx context.Context, resourceGroupName string, labName string, filter string, top *int32, orderBy string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}
	if len(filter) > 0 {
		queryParameters["$filter"] = autorest.Encode("query", filter)
	}
	if top != nil {
		queryParameters["$top"] = autorest.Encode("query", *top)
	}
	if len(orderBy) > 0 {
		queryParameters["$orderBy"] = autorest.Encode("query", orderBy)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) ListSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	return autorest.SendWithSender(client, req, sd...)
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client ScheduleClient) ListResponder(resp *http.Response) (result ResponseWithContinuationSchedule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// listNextResults retrieves the next set of results, if any.
func (client ScheduleClient) listNextResults(ctx context.Context, lastResults ResponseWithContinuationSchedule) (result ResponseWithContinuationSchedule, err error) {
	req, err := lastResults.responseWithContinuationSchedulePreparer(ctx)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "dtl.ScheduleClient", "listNextResults", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}
	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "dtl.ScheduleClient", "listNextResults", resp, "Failure sending next results request")
	}
	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "listNextResults", resp, "Failure responding to next results request")
	}
	return
}

// ListComplete enumerates all values, automatically crossing page boundaries as required.
func (client ScheduleClient) ListComplete(ctx context.Context, resourceGroupName string, labName string, filter string, top *int32, orderBy string) (result ResponseWithContinuationScheduleIterator, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.List")
		defer func() {
			sc := -1
			if result.Response().Response.Response != nil {
				sc = result.page.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	result.page, err = client.List(ctx, resourceGroupName, labName, filter, top, orderBy)
	return
}

// PatchResource modify properties of schedules.
// Parameters:
// resourceGroupName - the name of the resource group.
// labName - the name of the lab.
// name - the name of the schedule.
func (client ScheduleClient) PatchResource(ctx context.Context, resourceGroupName string, labName string, name string, schedule Schedule) (result Schedule, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ScheduleClient.PatchResource")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.PatchResourcePreparer(ctx, resourceGroupName, labName, name, schedule)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "PatchResource", nil, "Failure preparing request")
		return
	}

	resp, err := client.PatchResourceSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "PatchResource", resp, "Failure sending request")
		return
	}

	result, err = client.PatchResourceResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "dtl.ScheduleClient", "PatchResource", resp, "Failure responding to request")
	}

	return
}

// PatchResourcePreparer prepares the PatchResource request.
func (client ScheduleClient) PatchResourcePreparer(ctx context.Context, resourceGroupName string, labName string, name string, schedule Schedule) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"labName":           autorest.Encode("path", labName),
		"name":              autorest.Encode("path", name),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2015-05-21-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPatch(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}", pathParameters),
		autorest.WithJSON(schedule),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PatchResourceSender sends the PatchResource request. The method will close the
// http.Response Body if it receives an error.
func (client ScheduleClient) PatchResourceSender(req *http.Request) (*http.Response, error) {
	sd := autorest.GetSendDecorators(req.Context(), azure.DoRetryWithRegistration(client.Client))
	return autorest.SendWithSender(client, req, sd...)
}

// PatchResourceResponder handles the response to the PatchResource request. The method always
// closes the http.Response Body.
func (client ScheduleClient) PatchResourceResponder(resp *http.Response) (result Schedule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
