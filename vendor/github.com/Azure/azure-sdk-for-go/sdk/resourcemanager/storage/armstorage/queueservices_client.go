// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// Code generated by Microsoft (R) AutoRest Code Generator. DO NOT EDIT.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

package armstorage

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"net/http"
	"net/url"
	"strings"
)

// QueueServicesClient contains the methods for the QueueServices group.
// Don't use this type directly, use NewQueueServicesClient() instead.
type QueueServicesClient struct {
	internal       *arm.Client
	subscriptionID string
}

// NewQueueServicesClient creates a new instance of QueueServicesClient with the specified values.
//   - subscriptionID - The ID of the target subscription.
//   - credential - used to authorize requests. Usually a credential from azidentity.
//   - options - pass nil to accept the default values.
func NewQueueServicesClient(subscriptionID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*QueueServicesClient, error) {
	cl, err := arm.NewClient(moduleName, moduleVersion, credential, options)
	if err != nil {
		return nil, err
	}
	client := &QueueServicesClient{
		subscriptionID: subscriptionID,
		internal:       cl,
	}
	return client, nil
}

// GetServiceProperties - Gets the properties of a storage account’s Queue service, including properties for Storage Analytics
// and CORS (Cross-Origin Resource Sharing) rules.
// If the operation fails it returns an *azcore.ResponseError type.
//
// Generated from API version 2024-01-01
//   - resourceGroupName - The name of the resource group within the user's subscription. The name is case insensitive.
//   - accountName - The name of the storage account within the specified resource group. Storage account names must be between
//     3 and 24 characters in length and use numbers and lower-case letters only.
//   - options - QueueServicesClientGetServicePropertiesOptions contains the optional parameters for the QueueServicesClient.GetServiceProperties
//     method.
func (client *QueueServicesClient) GetServiceProperties(ctx context.Context, resourceGroupName string, accountName string, options *QueueServicesClientGetServicePropertiesOptions) (QueueServicesClientGetServicePropertiesResponse, error) {
	var err error
	const operationName = "QueueServicesClient.GetServiceProperties"
	ctx = context.WithValue(ctx, runtime.CtxAPINameKey{}, operationName)
	ctx, endSpan := runtime.StartSpan(ctx, operationName, client.internal.Tracer(), nil)
	defer func() { endSpan(err) }()
	req, err := client.getServicePropertiesCreateRequest(ctx, resourceGroupName, accountName, options)
	if err != nil {
		return QueueServicesClientGetServicePropertiesResponse{}, err
	}
	httpResp, err := client.internal.Pipeline().Do(req)
	if err != nil {
		return QueueServicesClientGetServicePropertiesResponse{}, err
	}
	if !runtime.HasStatusCode(httpResp, http.StatusOK) {
		err = runtime.NewResponseError(httpResp)
		return QueueServicesClientGetServicePropertiesResponse{}, err
	}
	resp, err := client.getServicePropertiesHandleResponse(httpResp)
	return resp, err
}

// getServicePropertiesCreateRequest creates the GetServiceProperties request.
func (client *QueueServicesClient) getServicePropertiesCreateRequest(ctx context.Context, resourceGroupName string, accountName string, _ *QueueServicesClientGetServicePropertiesOptions) (*policy.Request, error) {
	urlPath := "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}/queueServices/{queueServiceName}"
	if resourceGroupName == "" {
		return nil, errors.New("parameter resourceGroupName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{resourceGroupName}", url.PathEscape(resourceGroupName))
	if accountName == "" {
		return nil, errors.New("parameter accountName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{accountName}", url.PathEscape(accountName))
	if client.subscriptionID == "" {
		return nil, errors.New("parameter client.subscriptionID cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(client.subscriptionID))
	urlPath = strings.ReplaceAll(urlPath, "{queueServiceName}", url.PathEscape("default"))
	req, err := runtime.NewRequest(ctx, http.MethodGet, runtime.JoinPaths(client.internal.Endpoint(), urlPath))
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", "2024-01-01")
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header["Accept"] = []string{"application/json"}
	return req, nil
}

// getServicePropertiesHandleResponse handles the GetServiceProperties response.
func (client *QueueServicesClient) getServicePropertiesHandleResponse(resp *http.Response) (QueueServicesClientGetServicePropertiesResponse, error) {
	result := QueueServicesClientGetServicePropertiesResponse{}
	if err := runtime.UnmarshalAsJSON(resp, &result.QueueServiceProperties); err != nil {
		return QueueServicesClientGetServicePropertiesResponse{}, err
	}
	return result, nil
}

// List - List all queue services for the storage account
// If the operation fails it returns an *azcore.ResponseError type.
//
// Generated from API version 2024-01-01
//   - resourceGroupName - The name of the resource group within the user's subscription. The name is case insensitive.
//   - accountName - The name of the storage account within the specified resource group. Storage account names must be between
//     3 and 24 characters in length and use numbers and lower-case letters only.
//   - options - QueueServicesClientListOptions contains the optional parameters for the QueueServicesClient.List method.
func (client *QueueServicesClient) List(ctx context.Context, resourceGroupName string, accountName string, options *QueueServicesClientListOptions) (QueueServicesClientListResponse, error) {
	var err error
	const operationName = "QueueServicesClient.List"
	ctx = context.WithValue(ctx, runtime.CtxAPINameKey{}, operationName)
	ctx, endSpan := runtime.StartSpan(ctx, operationName, client.internal.Tracer(), nil)
	defer func() { endSpan(err) }()
	req, err := client.listCreateRequest(ctx, resourceGroupName, accountName, options)
	if err != nil {
		return QueueServicesClientListResponse{}, err
	}
	httpResp, err := client.internal.Pipeline().Do(req)
	if err != nil {
		return QueueServicesClientListResponse{}, err
	}
	if !runtime.HasStatusCode(httpResp, http.StatusOK) {
		err = runtime.NewResponseError(httpResp)
		return QueueServicesClientListResponse{}, err
	}
	resp, err := client.listHandleResponse(httpResp)
	return resp, err
}

// listCreateRequest creates the List request.
func (client *QueueServicesClient) listCreateRequest(ctx context.Context, resourceGroupName string, accountName string, _ *QueueServicesClientListOptions) (*policy.Request, error) {
	urlPath := "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}/queueServices"
	if resourceGroupName == "" {
		return nil, errors.New("parameter resourceGroupName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{resourceGroupName}", url.PathEscape(resourceGroupName))
	if accountName == "" {
		return nil, errors.New("parameter accountName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{accountName}", url.PathEscape(accountName))
	if client.subscriptionID == "" {
		return nil, errors.New("parameter client.subscriptionID cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(client.subscriptionID))
	req, err := runtime.NewRequest(ctx, http.MethodGet, runtime.JoinPaths(client.internal.Endpoint(), urlPath))
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", "2024-01-01")
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header["Accept"] = []string{"application/json"}
	return req, nil
}

// listHandleResponse handles the List response.
func (client *QueueServicesClient) listHandleResponse(resp *http.Response) (QueueServicesClientListResponse, error) {
	result := QueueServicesClientListResponse{}
	if err := runtime.UnmarshalAsJSON(resp, &result.ListQueueServices); err != nil {
		return QueueServicesClientListResponse{}, err
	}
	return result, nil
}

// SetServiceProperties - Sets the properties of a storage account’s Queue service, including properties for Storage Analytics
// and CORS (Cross-Origin Resource Sharing) rules.
// If the operation fails it returns an *azcore.ResponseError type.
//
// Generated from API version 2024-01-01
//   - resourceGroupName - The name of the resource group within the user's subscription. The name is case insensitive.
//   - accountName - The name of the storage account within the specified resource group. Storage account names must be between
//     3 and 24 characters in length and use numbers and lower-case letters only.
//   - parameters - The properties of a storage account’s Queue service, only properties for Storage Analytics and CORS (Cross-Origin
//     Resource Sharing) rules can be specified.
//   - options - QueueServicesClientSetServicePropertiesOptions contains the optional parameters for the QueueServicesClient.SetServiceProperties
//     method.
func (client *QueueServicesClient) SetServiceProperties(ctx context.Context, resourceGroupName string, accountName string, parameters QueueServiceProperties, options *QueueServicesClientSetServicePropertiesOptions) (QueueServicesClientSetServicePropertiesResponse, error) {
	var err error
	const operationName = "QueueServicesClient.SetServiceProperties"
	ctx = context.WithValue(ctx, runtime.CtxAPINameKey{}, operationName)
	ctx, endSpan := runtime.StartSpan(ctx, operationName, client.internal.Tracer(), nil)
	defer func() { endSpan(err) }()
	req, err := client.setServicePropertiesCreateRequest(ctx, resourceGroupName, accountName, parameters, options)
	if err != nil {
		return QueueServicesClientSetServicePropertiesResponse{}, err
	}
	httpResp, err := client.internal.Pipeline().Do(req)
	if err != nil {
		return QueueServicesClientSetServicePropertiesResponse{}, err
	}
	if !runtime.HasStatusCode(httpResp, http.StatusOK) {
		err = runtime.NewResponseError(httpResp)
		return QueueServicesClientSetServicePropertiesResponse{}, err
	}
	resp, err := client.setServicePropertiesHandleResponse(httpResp)
	return resp, err
}

// setServicePropertiesCreateRequest creates the SetServiceProperties request.
func (client *QueueServicesClient) setServicePropertiesCreateRequest(ctx context.Context, resourceGroupName string, accountName string, parameters QueueServiceProperties, _ *QueueServicesClientSetServicePropertiesOptions) (*policy.Request, error) {
	urlPath := "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}/queueServices/{queueServiceName}"
	if resourceGroupName == "" {
		return nil, errors.New("parameter resourceGroupName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{resourceGroupName}", url.PathEscape(resourceGroupName))
	if accountName == "" {
		return nil, errors.New("parameter accountName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{accountName}", url.PathEscape(accountName))
	if client.subscriptionID == "" {
		return nil, errors.New("parameter client.subscriptionID cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{subscriptionId}", url.PathEscape(client.subscriptionID))
	urlPath = strings.ReplaceAll(urlPath, "{queueServiceName}", url.PathEscape("default"))
	req, err := runtime.NewRequest(ctx, http.MethodPut, runtime.JoinPaths(client.internal.Endpoint(), urlPath))
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", "2024-01-01")
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header["Accept"] = []string{"application/json"}
	if err := runtime.MarshalAsJSON(req, parameters); err != nil {
		return nil, err
	}
	return req, nil
}

// setServicePropertiesHandleResponse handles the SetServiceProperties response.
func (client *QueueServicesClient) setServicePropertiesHandleResponse(resp *http.Response) (QueueServicesClientSetServicePropertiesResponse, error) {
	result := QueueServicesClientSetServicePropertiesResponse{}
	if err := runtime.UnmarshalAsJSON(resp, &result.QueueServiceProperties); err != nil {
		return QueueServicesClientSetServicePropertiesResponse{}, err
	}
	return result, nil
}
