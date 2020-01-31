// +build go1.9

// Copyright 2020 Microsoft Corporation
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

package privatedns

import (
	"context"

	original "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
)

const (
	DefaultBaseURI = original.DefaultBaseURI
)

type ProvisioningState = original.ProvisioningState

const (
	Canceled  ProvisioningState = original.Canceled
	Creating  ProvisioningState = original.Creating
	Deleting  ProvisioningState = original.Deleting
	Failed    ProvisioningState = original.Failed
	Succeeded ProvisioningState = original.Succeeded
	Updating  ProvisioningState = original.Updating
)

type RecordType = original.RecordType

const (
	A     RecordType = original.A
	AAAA  RecordType = original.AAAA
	CNAME RecordType = original.CNAME
	MX    RecordType = original.MX
	PTR   RecordType = original.PTR
	SOA   RecordType = original.SOA
	SRV   RecordType = original.SRV
	TXT   RecordType = original.TXT
)

type VirtualNetworkLinkState = original.VirtualNetworkLinkState

const (
	Completed  VirtualNetworkLinkState = original.Completed
	InProgress VirtualNetworkLinkState = original.InProgress
)

type ARecord = original.ARecord
type AaaaRecord = original.AaaaRecord
type BaseClient = original.BaseClient
type CloudError = original.CloudError
type CloudErrorBody = original.CloudErrorBody
type CnameRecord = original.CnameRecord
type MxRecord = original.MxRecord
type PrivateZone = original.PrivateZone
type PrivateZoneListResult = original.PrivateZoneListResult
type PrivateZoneListResultIterator = original.PrivateZoneListResultIterator
type PrivateZoneListResultPage = original.PrivateZoneListResultPage
type PrivateZoneProperties = original.PrivateZoneProperties
type PrivateZonesClient = original.PrivateZonesClient
type PrivateZonesCreateOrUpdateFuture = original.PrivateZonesCreateOrUpdateFuture
type PrivateZonesDeleteFuture = original.PrivateZonesDeleteFuture
type PrivateZonesUpdateFuture = original.PrivateZonesUpdateFuture
type ProxyResource = original.ProxyResource
type PtrRecord = original.PtrRecord
type RecordSet = original.RecordSet
type RecordSetListResult = original.RecordSetListResult
type RecordSetListResultIterator = original.RecordSetListResultIterator
type RecordSetListResultPage = original.RecordSetListResultPage
type RecordSetProperties = original.RecordSetProperties
type RecordSetsClient = original.RecordSetsClient
type Resource = original.Resource
type SoaRecord = original.SoaRecord
type SrvRecord = original.SrvRecord
type SubResource = original.SubResource
type TrackedResource = original.TrackedResource
type TxtRecord = original.TxtRecord
type VirtualNetworkLink = original.VirtualNetworkLink
type VirtualNetworkLinkListResult = original.VirtualNetworkLinkListResult
type VirtualNetworkLinkListResultIterator = original.VirtualNetworkLinkListResultIterator
type VirtualNetworkLinkListResultPage = original.VirtualNetworkLinkListResultPage
type VirtualNetworkLinkProperties = original.VirtualNetworkLinkProperties
type VirtualNetworkLinksClient = original.VirtualNetworkLinksClient
type VirtualNetworkLinksCreateOrUpdateFuture = original.VirtualNetworkLinksCreateOrUpdateFuture
type VirtualNetworkLinksDeleteFuture = original.VirtualNetworkLinksDeleteFuture
type VirtualNetworkLinksUpdateFuture = original.VirtualNetworkLinksUpdateFuture

func New(subscriptionID string) BaseClient {
	return original.New(subscriptionID)
}
func NewPrivateZoneListResultIterator(page PrivateZoneListResultPage) PrivateZoneListResultIterator {
	return original.NewPrivateZoneListResultIterator(page)
}
func NewPrivateZoneListResultPage(getNextPage func(context.Context, PrivateZoneListResult) (PrivateZoneListResult, error)) PrivateZoneListResultPage {
	return original.NewPrivateZoneListResultPage(getNextPage)
}
func NewPrivateZonesClient(subscriptionID string) PrivateZonesClient {
	return original.NewPrivateZonesClient(subscriptionID)
}
func NewPrivateZonesClientWithBaseURI(baseURI string, subscriptionID string) PrivateZonesClient {
	return original.NewPrivateZonesClientWithBaseURI(baseURI, subscriptionID)
}
func NewRecordSetListResultIterator(page RecordSetListResultPage) RecordSetListResultIterator {
	return original.NewRecordSetListResultIterator(page)
}
func NewRecordSetListResultPage(getNextPage func(context.Context, RecordSetListResult) (RecordSetListResult, error)) RecordSetListResultPage {
	return original.NewRecordSetListResultPage(getNextPage)
}
func NewRecordSetsClient(subscriptionID string) RecordSetsClient {
	return original.NewRecordSetsClient(subscriptionID)
}
func NewRecordSetsClientWithBaseURI(baseURI string, subscriptionID string) RecordSetsClient {
	return original.NewRecordSetsClientWithBaseURI(baseURI, subscriptionID)
}
func NewVirtualNetworkLinkListResultIterator(page VirtualNetworkLinkListResultPage) VirtualNetworkLinkListResultIterator {
	return original.NewVirtualNetworkLinkListResultIterator(page)
}
func NewVirtualNetworkLinkListResultPage(getNextPage func(context.Context, VirtualNetworkLinkListResult) (VirtualNetworkLinkListResult, error)) VirtualNetworkLinkListResultPage {
	return original.NewVirtualNetworkLinkListResultPage(getNextPage)
}
func NewVirtualNetworkLinksClient(subscriptionID string) VirtualNetworkLinksClient {
	return original.NewVirtualNetworkLinksClient(subscriptionID)
}
func NewVirtualNetworkLinksClientWithBaseURI(baseURI string, subscriptionID string) VirtualNetworkLinksClient {
	return original.NewVirtualNetworkLinksClientWithBaseURI(baseURI, subscriptionID)
}
func NewWithBaseURI(baseURI string, subscriptionID string) BaseClient {
	return original.NewWithBaseURI(baseURI, subscriptionID)
}
func PossibleProvisioningStateValues() []ProvisioningState {
	return original.PossibleProvisioningStateValues()
}
func PossibleRecordTypeValues() []RecordType {
	return original.PossibleRecordTypeValues()
}
func PossibleVirtualNetworkLinkStateValues() []VirtualNetworkLinkState {
	return original.PossibleVirtualNetworkLinkStateValues()
}
func UserAgent() string {
	return original.UserAgent() + " profiles/latest"
}
func Version() string {
	return original.Version()
}
