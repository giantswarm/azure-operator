// +build go1.9

// Copyright 2017 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code was auto-generated by:
// github.com/Azure/azure-sdk-for-go/tools/profileBuilder
// commit ID: dab57ee609fffdc578f48519d5cdc980efd8cc00

package storage

import original "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2016-01-01/storage"

type AccountsClient = original.AccountsClient

const (
	DefaultBaseURI = original.DefaultBaseURI
)

type ManagementClient = original.ManagementClient
type AccessTier = original.AccessTier

const (
	Cool AccessTier = original.Cool
	Hot  AccessTier = original.Hot
)

type AccountStatus = original.AccountStatus

const (
	Available   AccountStatus = original.Available
	Unavailable AccountStatus = original.Unavailable
)

type KeyPermission = original.KeyPermission

const (
	FULL KeyPermission = original.FULL
	READ KeyPermission = original.READ
)

type Kind = original.Kind

const (
	BlobStorage Kind = original.BlobStorage
	Storage     Kind = original.Storage
)

type ProvisioningState = original.ProvisioningState

const (
	Creating     ProvisioningState = original.Creating
	ResolvingDNS ProvisioningState = original.ResolvingDNS
	Succeeded    ProvisioningState = original.Succeeded
)

type Reason = original.Reason

const (
	AccountNameInvalid Reason = original.AccountNameInvalid
	AlreadyExists      Reason = original.AlreadyExists
)

type SkuName = original.SkuName

const (
	PremiumLRS    SkuName = original.PremiumLRS
	StandardGRS   SkuName = original.StandardGRS
	StandardLRS   SkuName = original.StandardLRS
	StandardRAGRS SkuName = original.StandardRAGRS
	StandardZRS   SkuName = original.StandardZRS
)

type SkuTier = original.SkuTier

const (
	Premium  SkuTier = original.Premium
	Standard SkuTier = original.Standard
)

type UsageUnit = original.UsageUnit

const (
	Bytes           UsageUnit = original.Bytes
	BytesPerSecond  UsageUnit = original.BytesPerSecond
	Count           UsageUnit = original.Count
	CountsPerSecond UsageUnit = original.CountsPerSecond
	Percent         UsageUnit = original.Percent
	Seconds         UsageUnit = original.Seconds
)

type Account = original.Account
type AccountCheckNameAvailabilityParameters = original.AccountCheckNameAvailabilityParameters
type AccountCreateParameters = original.AccountCreateParameters
type AccountKey = original.AccountKey
type AccountListKeysResult = original.AccountListKeysResult
type AccountListResult = original.AccountListResult
type AccountProperties = original.AccountProperties
type AccountPropertiesCreateParameters = original.AccountPropertiesCreateParameters
type AccountPropertiesUpdateParameters = original.AccountPropertiesUpdateParameters
type AccountRegenerateKeyParameters = original.AccountRegenerateKeyParameters
type AccountUpdateParameters = original.AccountUpdateParameters
type CheckNameAvailabilityResult = original.CheckNameAvailabilityResult
type CustomDomain = original.CustomDomain
type Encryption = original.Encryption
type EncryptionService = original.EncryptionService
type EncryptionServices = original.EncryptionServices
type Endpoints = original.Endpoints
type Resource = original.Resource
type Sku = original.Sku
type Usage = original.Usage
type UsageListResult = original.UsageListResult
type UsageName = original.UsageName
type UsageClient = original.UsageClient

func NewAccountsClient(subscriptionID string) AccountsClient {
	return original.NewAccountsClient(subscriptionID)
}
func NewAccountsClientWithBaseURI(baseURI string, subscriptionID string) AccountsClient {
	return original.NewAccountsClientWithBaseURI(baseURI, subscriptionID)
}
func New(subscriptionID string) ManagementClient {
	return original.New(subscriptionID)
}
func NewWithBaseURI(baseURI string, subscriptionID string) ManagementClient {
	return original.NewWithBaseURI(baseURI, subscriptionID)
}
func NewUsageClient(subscriptionID string) UsageClient {
	return original.NewUsageClient(subscriptionID)
}
func NewUsageClientWithBaseURI(baseURI string, subscriptionID string) UsageClient {
	return original.NewUsageClientWithBaseURI(baseURI, subscriptionID)
}
func UserAgent() string {
	return original.UserAgent() + " profiles/2017-03-09"
}
func Version() string {
	return original.Version()
}
