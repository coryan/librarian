// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cpp

import (
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
)

func hasStreamingMethod(methods []*api.Method) bool {
	return hasStreamingReadMethod(methods) || hasStreamingWriteMethod(methods) || hasBidiStreamingMethod(methods)
}

func printDecoratorPublicMethods(p *printer, svc *api.Service, methods, asyncMethods []*api.Method, serviceVars map[string]string, lib *config.Library, model *api.API) {
	for _, m := range methods {
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		if isStreamingWrite(m) {
			p.PrintWith(mVars, `
  std::unique_ptr<::google::cloud::internal::StreamingWriteRpc<
      $request_type$,
      $response_type$>>
  $method_name$(
      std::shared_ptr<grpc::ClientContext> context,
      Options const& options) override;
`)
			continue
		}
		if isBidiStreaming(m) {
			p.PrintWith(mVars, `
  std::unique_ptr<::google::cloud::AsyncStreamingReadWriteRpc<
      $request_type$,
      $response_type$>>
  Async$method_name$(
      google::cloud::CompletionQueue const& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options) override;
`)
			continue
		}
		if isLongrunning(m) {
			p.PrintWith(mVars, `
  future<StatusOr<google::longrunning::Operation>> Async$method_name$(
      google::cloud::CompletionQueue& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      $request_type$ const& request) override;

  StatusOr<google::longrunning::Operation> $method_name$(
      grpc::ClientContext& context,
      Options options,
      $request_type$ const& request) override;
`)
			continue
		}
		if isStreamingRead(m) {
			p.PrintWith(mVars, `
  std::unique_ptr<google::cloud::internal::StreamingReadRpc<$response_type$>>
  $method_name$(
      std::shared_ptr<grpc::ClientContext> context,
      Options const& options,
      $request_type$ const& request) override;
`)
			continue
		}
		p.PrintWith(mVars, `
  $return_type$ $method_name$(
      grpc::ClientContext& context,
      Options const& options,
      $request_type$ const& request) override;
`)
	}

	for _, m := range asyncMethods {
		if isBidiStreaming(m) || isLongrunning(m) {
			continue
		}
		mVars := buildMethodVars(svc, m, serviceVars, lib, model)
		if isStreamingRead(m) {
			p.PrintWith(mVars, `
  std::unique_ptr<::google::cloud::internal::AsyncStreamingReadRpc<
      $response_type$>>
  Async$method_name$(
      google::cloud::CompletionQueue const& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      $request_type$ const& request) override;
`)
			continue
		}
		if isStreamingWrite(m) {
			p.PrintWith(mVars, `
  std::unique_ptr<::google::cloud::internal::AsyncStreamingWriteRpc<
      $request_type$, $response_type$>>
  Async$method_name$(
      google::cloud::CompletionQueue const& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options) override;
`)
			continue
		}
		p.PrintWith(mVars, `
  future<$return_type$> Async$method_name$(
      google::cloud::CompletionQueue& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      $request_type$ const& request) override;
`)
	}

	if hasLongrunningMethod(methods) {
		p.Print(`
  future<StatusOr<google::longrunning::Operation>> AsyncGetOperation(
      google::cloud::CompletionQueue& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      google::longrunning::GetOperationRequest const& request) override;

  future<Status> AsyncCancelOperation(
      google::cloud::CompletionQueue& cq,
      std::shared_ptr<grpc::ClientContext> context,
      google::cloud::internal::ImmutableOptions options,
      google::longrunning::CancelOperationRequest const& request) override;
`)
	}
}
