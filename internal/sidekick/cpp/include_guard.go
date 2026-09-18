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
	"path/filepath"
	"strings"
)

// formatHeaderIncludeGuard derives the C++ header include guard from the header's
// relative include path, matching the google-cloud-cpp generator convention.
// For example:
//
//	"google/cloud/golden/v1/golden_kitchen_sink_client.h"
//	-> "GOOGLE_CLOUD_CPP_GOOGLE_CLOUD_GOLDEN_V1_GOLDEN_KITCHEN_SINK_CLIENT_H"
func formatHeaderIncludeGuard(headerPath string) string {
	clean := strings.ReplaceAll(headerPath, "\\", "/")
	clean = strings.TrimPrefix(clean, "/")
	s := "GOOGLE_CLOUD_CPP_" + clean
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToUpper(s)
}

func (ann *serviceAnnotations) ClientHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ClientHeader()))
}

func (ann *serviceAnnotations) ClientHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ClientHeaderPath())
}

func (ann *serviceAnnotations) ConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ConnectionHeader()))
}

func (ann *serviceAnnotations) ConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ConnectionHeaderPath())
}

func (ann *serviceAnnotations) IdempotencyPolicyHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.IdempotencyPolicyHeader()))
}

func (ann *serviceAnnotations) IdempotencyPolicyHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.IdempotencyPolicyHeaderPath())
}

func (ann *serviceAnnotations) OptionsHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.OptionsHeader()))
}

func (ann *serviceAnnotations) OptionsHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.OptionsHeaderPath())
}

func (ann *serviceAnnotations) RestConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestConnectionHeader()))
}

func (ann *serviceAnnotations) RestConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestConnectionHeaderPath())
}

func (ann *serviceAnnotations) MockConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.MockConnectionHeader()))
}

func (ann *serviceAnnotations) MockConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.MockConnectionHeaderPath())
}

func (ann *serviceAnnotations) OptionDefaultsHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.OptionDefaultsHeader()))
}

func (ann *serviceAnnotations) OptionDefaultsHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.OptionDefaultsHeaderPath())
}

func (ann *serviceAnnotations) TracingConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.TracingConnectionHeader()))
}

func (ann *serviceAnnotations) TracingConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.TracingConnectionHeaderPath())
}

func (ann *serviceAnnotations) RetryTraitsHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RetryTraitsHeader()))
}

func (ann *serviceAnnotations) RetryTraitsHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RetryTraitsHeaderPath())
}

func (ann *serviceAnnotations) ConnectionImplHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.ConnectionImplHeader()))
}

func (ann *serviceAnnotations) ConnectionImplHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ConnectionImplHeaderPath())
}

func (ann *serviceAnnotations) StubFactoryHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.StubFactoryHeader()))
}

func (ann *serviceAnnotations) StubFactoryHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.StubFactoryHeaderPath())
}

func (ann *serviceAnnotations) AuthDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.AuthDecoratorHeader()))
}

func (ann *serviceAnnotations) AuthDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.AuthDecoratorHeaderPath())
}

func (ann *serviceAnnotations) LoggingDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.LoggingDecoratorHeader()))
}

func (ann *serviceAnnotations) LoggingDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.LoggingDecoratorHeaderPath())
}

func (ann *serviceAnnotations) MetadataDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.MetadataDecoratorHeader()))
}

func (ann *serviceAnnotations) MetadataDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.MetadataDecoratorHeaderPath())
}

func (ann *serviceAnnotations) StubHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.StubHeader()))
}

func (ann *serviceAnnotations) StubHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.StubHeaderPath())
}

func (ann *serviceAnnotations) TracingStubHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.TracingStubHeader()))
}

func (ann *serviceAnnotations) TracingStubHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.TracingStubHeaderPath())
}

func (ann *serviceAnnotations) RoundRobinDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RoundRobinDecoratorHeader()))
}

func (ann *serviceAnnotations) RoundRobinDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RoundRobinDecoratorHeaderPath())
}

func (ann *serviceAnnotations) RestConnectionImplHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestConnectionImplHeader()))
}

func (ann *serviceAnnotations) RestConnectionImplHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestConnectionImplHeaderPath())
}

func (ann *serviceAnnotations) RestLoggingDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestLoggingDecoratorHeader()))
}

func (ann *serviceAnnotations) RestLoggingDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestLoggingDecoratorHeaderPath())
}

func (ann *serviceAnnotations) RestMetadataDecoratorHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestMetadataDecoratorHeader()))
}

func (ann *serviceAnnotations) RestMetadataDecoratorHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestMetadataDecoratorHeaderPath())
}

func (ann *serviceAnnotations) RestStubHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestStubHeader()))
}

func (ann *serviceAnnotations) RestStubHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestStubHeaderPath())
}

func (ann *serviceAnnotations) RestStubFactoryHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ProductPath, ann.RestStubFactoryHeader()))
}

func (ann *serviceAnnotations) RestStubFactoryHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.RestStubFactoryHeaderPath())
}

func (ann *serviceAnnotations) ForwardingClientHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ForwardingPath, ann.ClientHeader()))
}

func (ann *serviceAnnotations) ForwardingClientHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ForwardingClientHeaderPath())
}

func (ann *serviceAnnotations) ForwardingConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ForwardingPath, ann.ConnectionHeader()))
}

func (ann *serviceAnnotations) ForwardingConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ForwardingConnectionHeaderPath())
}

func (ann *serviceAnnotations) ForwardingIdempotencyPolicyHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ForwardingPath, ann.IdempotencyPolicyHeader()))
}

func (ann *serviceAnnotations) ForwardingIdempotencyPolicyHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ForwardingIdempotencyPolicyHeaderPath())
}

func (ann *serviceAnnotations) ForwardingOptionsHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ForwardingPath, ann.OptionsHeader()))
}

func (ann *serviceAnnotations) ForwardingOptionsHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ForwardingOptionsHeaderPath())
}

func (ann *serviceAnnotations) ForwardingMockConnectionHeaderPath() string {
	return filepath.ToSlash(filepath.Join(ann.ForwardingPath, ann.MockConnectionHeader()))
}

func (ann *serviceAnnotations) ForwardingMockConnectionHeaderGuard() string {
	return formatHeaderIncludeGuard(ann.ForwardingMockConnectionHeaderPath())
}
