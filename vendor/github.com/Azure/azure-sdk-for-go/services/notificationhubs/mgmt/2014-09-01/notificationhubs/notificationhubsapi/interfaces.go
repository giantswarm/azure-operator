package notificationhubsapi

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
	"github.com/Azure/azure-sdk-for-go/services/notificationhubs/mgmt/2014-09-01/notificationhubs"
	"github.com/Azure/go-autorest/autorest"
)

// NamespacesClientAPI contains the set of methods on the NamespacesClient type.
type NamespacesClientAPI interface {
	CheckAvailability(ctx context.Context, parameters notificationhubs.CheckAvailabilityParameters) (result notificationhubs.CheckAvailabilityResource, err error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, namespaceName string, parameters notificationhubs.NamespaceCreateOrUpdateParameters) (result notificationhubs.NamespaceResource, err error)
	CreateOrUpdateAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, authorizationRuleName string, parameters notificationhubs.SharedAccessAuthorizationRuleCreateOrUpdateParameters) (result notificationhubs.SharedAccessAuthorizationRuleResource, err error)
	Delete(ctx context.Context, resourceGroupName string, namespaceName string) (result notificationhubs.NamespacesDeleteFuture, err error)
	DeleteAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, authorizationRuleName string) (result autorest.Response, err error)
	Get(ctx context.Context, resourceGroupName string, namespaceName string) (result notificationhubs.NamespaceResource, err error)
	GetAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, authorizationRuleName string) (result notificationhubs.SharedAccessAuthorizationRuleResource, err error)
	GetLongRunningOperationStatus(ctx context.Context, operationStatusLink string) (result autorest.Response, err error)
	List(ctx context.Context, resourceGroupName string) (result notificationhubs.NamespaceListResultPage, err error)
	ListAll(ctx context.Context) (result notificationhubs.NamespaceListResultPage, err error)
	ListAuthorizationRules(ctx context.Context, resourceGroupName string, namespaceName string) (result notificationhubs.SharedAccessAuthorizationRuleListResultPage, err error)
	ListKeys(ctx context.Context, resourceGroupName string, namespaceName string, authorizationRuleName string) (result notificationhubs.ResourceListKeys, err error)
}

var _ NamespacesClientAPI = (*notificationhubs.NamespacesClient)(nil)

// ClientAPI contains the set of methods on the Client type.
type ClientAPI interface {
	CheckAvailability(ctx context.Context, resourceGroupName string, namespaceName string, parameters notificationhubs.CheckAvailabilityParameters) (result notificationhubs.CheckAvailabilityResource, err error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string, parameters notificationhubs.CreateOrUpdateParameters) (result notificationhubs.ResourceType, err error)
	CreateOrUpdateAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string, authorizationRuleName string, parameters notificationhubs.SharedAccessAuthorizationRuleCreateOrUpdateParameters) (result notificationhubs.SharedAccessAuthorizationRuleResource, err error)
	Delete(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string) (result autorest.Response, err error)
	DeleteAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string, authorizationRuleName string) (result autorest.Response, err error)
	Get(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string) (result notificationhubs.ResourceType, err error)
	GetAuthorizationRule(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string, authorizationRuleName string) (result notificationhubs.SharedAccessAuthorizationRuleResource, err error)
	GetPnsCredentials(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string) (result notificationhubs.ResourceType, err error)
	List(ctx context.Context, resourceGroupName string, namespaceName string) (result notificationhubs.ListResultPage, err error)
	ListAuthorizationRules(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string) (result notificationhubs.SharedAccessAuthorizationRuleListResultPage, err error)
	ListKeys(ctx context.Context, resourceGroupName string, namespaceName string, notificationHubName string, authorizationRuleName string) (result notificationhubs.ResourceListKeys, err error)
}

var _ ClientAPI = (*notificationhubs.Client)(nil)
