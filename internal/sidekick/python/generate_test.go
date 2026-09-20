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

package python

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/librarian/internal/config"
	"github.com/googleapis/librarian/internal/sidekick/api"
	"github.com/googleapis/librarian/internal/sidekick/language"
)

func TestGenerate(t *testing.T) {
	outdir := t.TempDir()

	msg := api.NewTestMessage("Secret").WithFields(
		api.NewTestField("name").WithType(api.TypezString),
	)
	svc := api.NewTestService("SecretManagerService")
	model := api.NewTestAPI([]*api.Message{msg}, nil, []*api.Service{svc})
	model.Name = "google-cloud-secretmanager"

	lib := &config.Library{
		Name:          "google-cloud-secretmanager",
		Version:       "2.1.0",
		CopyrightYear: "2026",
		APIs: []*config.API{
			{Path: "google/cloud/secretmanager/v1"},
		},
		Python: &config.PythonPackage{
			DefaultVersion: "v1",
		},
	}

	if err := Generate(t.Context(), model, outdir, lib); err != nil {
		t.Fatal(err)
	}

	// Verify gapic_version.py
	versionFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "gapic_version.py")
	versionBytes, err := os.ReadFile(versionFile)
	if err != nil {
		t.Fatal(err)
	}
	wantVersion := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
__version__ = "2.1.0"  # {x-release-please-version}
`
	if diff := cmp.Diff(wantVersion, string(versionBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify root package gapic_version.py
	rootVersionFile := filepath.Join(outdir, "google", "cloud", "secretmanager", "gapic_version.py")
	rootVersionBytes, err := os.ReadFile(rootVersionFile)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantVersion, string(rootVersionBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify py.typed
	typedFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "py.typed")
	typedBytes, err := os.ReadFile(typedFile)
	if err != nil {
		t.Fatal(err)
	}
	wantTyped := `# Marker file for PEP 561.
# The google-cloud-secretmanager package uses inline types.
`
	if diff := cmp.Diff(wantTyped, string(typedBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify root package py.typed
	rootTypedFile := filepath.Join(outdir, "google", "cloud", "secretmanager", "py.typed")
	rootTypedBytes, err := os.ReadFile(rootTypedFile)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(wantTyped, string(rootTypedBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify _compat.py
	compatFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "_compat.py")
	compatBytes, err := os.ReadFile(compatFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(compatBytes) == 0 {
		t.Errorf("_compat.py is empty")
	}

	// Verify services/__init__.py
	servicesInitFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "__init__.py")
	servicesInitBytes, err := os.ReadFile(servicesInitFile)
	if err != nil {
		t.Fatal(err)
	}
	wantServicesInit := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
`
	if diff := cmp.Diff(wantServicesInit, string(servicesInitBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/__init__.py
	svcInitFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "__init__.py")
	svcInitBytes, err := os.ReadFile(svcInitFile)
	if err != nil {
		t.Fatal(err)
	}
	wantSvcInit := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
from .client import SecretManagerServiceClient
from .async_client import SecretManagerServiceAsyncClient

__all__ = (
    'SecretManagerServiceClient',
    'SecretManagerServiceAsyncClient',
)
`
	if diff := cmp.Diff(wantSvcInit, string(svcInitBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/transports/README.rst
	readmeFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "README.rst")
	readmeBytes, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatal(err)
	}
	wantReadme := `
transport inheritance structure
_______________________________

` + "``SecretManagerServiceTransport`` is the ABC for all transports." + `

- public child ` + "``SecretManagerServiceGrpcTransport`` for sync gRPC transport (defined in ``grpc.py``)." + `
- public child ` + "``SecretManagerServiceGrpcAsyncIOTransport`` for async gRPC transport (defined in ``grpc_asyncio.py``)." + `
- private child ` + "``_BaseSecretManagerServiceRestTransport`` for base REST transport with inner classes ``_BaseMETHOD`` (defined in ``rest_base.py``)." + `
- public child ` + "``SecretManagerServiceRestTransport`` for sync REST transport with inner classes ``METHOD`` derived from the parent's corresponding ``_BaseMETHOD`` classes (defined in ``rest.py``)." + `
`
	if diff := cmp.Diff(wantReadme, string(readmeBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/transports/__init__.py
	transInitFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "__init__.py")
	transInitBytes, err := os.ReadFile(transInitFile)
	if err != nil {
		t.Fatal(err)
	}
	wantTransInit := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
from collections import OrderedDict
from typing import Dict, Type

from .base import SecretManagerServiceTransport
from .grpc import SecretManagerServiceGrpcTransport
from .grpc_asyncio import SecretManagerServiceGrpcAsyncIOTransport
from .rest import SecretManagerServiceRestTransport
from .rest import SecretManagerServiceRestInterceptor


# Compile a registry of transports.
_transport_registry = OrderedDict()  # type: Dict[str, Type[SecretManagerServiceTransport]]
_transport_registry['grpc'] = SecretManagerServiceGrpcTransport
_transport_registry['grpc_asyncio'] = SecretManagerServiceGrpcAsyncIOTransport
_transport_registry['rest'] = SecretManagerServiceRestTransport

__all__ = (
    'SecretManagerServiceTransport',
    'SecretManagerServiceGrpcTransport',
    'SecretManagerServiceGrpcAsyncIOTransport',
    'SecretManagerServiceRestTransport',
    'SecretManagerServiceRestInterceptor',
)
`
	if diff := cmp.Diff(wantTransInit, string(transInitBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/transports/base.py
	baseFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "base.py")
	baseBytes, err := os.ReadFile(baseFile)
	if err != nil {
		t.Fatal(err)
	}
	wantBase := `# -*- coding: utf-8 -*-
# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
import abc
from typing import Awaitable, Callable, Dict, Optional, Sequence, Union

from google.cloud.secretmanager_v1 import gapic_version as package_version

import google.auth  # type: ignore
import google.api_core
from google.api_core import exceptions as core_exceptions
from google.api_core import gapic_v1
from google.api_core import retry as retries
from google.auth import credentials as ga_credentials  # type: ignore
from google.oauth2 import service_account # type: ignore
import google.protobuf


DEFAULT_CLIENT_INFO = gapic_v1.client_info.ClientInfo(gapic_version=package_version.__version__)
DEFAULT_CLIENT_INFO.protobuf_runtime_version = google.protobuf.__version__


class SecretManagerServiceTransport(abc.ABC):
    """Abstract transport class for SecretManagerService."""

    AUTH_SCOPES = (
        'https://www.googleapis.com/auth/cloud-platform',
    )

    DEFAULT_HOST: str = ''

    def __init__(
            self, *,
            host: str = DEFAULT_HOST,
            credentials: Optional[ga_credentials.Credentials] = None,
            credentials_file: Optional[str] = None,
            scopes: Optional[Sequence[str]] = None,
            quota_project_id: Optional[str] = None,
            client_info: gapic_v1.client_info.ClientInfo = DEFAULT_CLIENT_INFO,
            always_use_jwt_access: Optional[bool] = False,
            api_audience: Optional[str] = None,
            **kwargs,
            ) -> None:
        """Instantiate the transport.

        Args:
            host (Optional[str]):
                 The hostname to connect to (default: '').
            credentials (Optional[google.auth.credentials.Credentials]): The
                authorization credentials to attach to requests. These
                credentials identify the application to the service; if none
                are specified, the client will attempt to ascertain the
                credentials from the environment.
            credentials_file (Optional[str]): Deprecated. A file with credentials that can
                be loaded with :func:` + "`google.auth.load_credentials_from_file`" + `.
                This argument is mutually exclusive with credentials. This argument will be
                removed in the next major version of this library.
            scopes (Optional[Sequence[str]]): A list of scopes.
            quota_project_id (Optional[str]): An optional project to use for billing
                and quota.
            client_info (google.api_core.gapic_v1.client_info.ClientInfo):
                The client info used to send a user-agent string along with
                API requests. If ` + "``None``" + `, then default info will be used.
                Generally, you only need to set this if you're developing
                your own client library.
            always_use_jwt_access (Optional[bool]): Whether self signed JWT should
                be used for service account credentials.
            api_audience (Optional[str]): The intended audience for the API calls
                to the service that will be set when using certain 3rd party
                authentication flows. Audience is typically a resource identifier.
                If not set, the host value will be used as a default.
        """

        # Save the scopes.
        self._scopes = scopes
        if not hasattr(self, "_ignore_credentials"):
            self._ignore_credentials: bool = False

        # If no credentials are provided, then determine the appropriate
        # defaults.
        if credentials and credentials_file:
            raise core_exceptions.DuplicateCredentialArgs("'credentials_file' and 'credentials' are mutually exclusive")

        if credentials_file is not None:
            credentials, _ = google.auth.load_credentials_from_file(
                                credentials_file,
                                scopes=scopes,
                                quota_project_id=quota_project_id,
                                default_scopes=self.AUTH_SCOPES,
                            )
        elif credentials is None and not self._ignore_credentials:
            credentials, _ = google.auth.default(scopes=scopes, quota_project_id=quota_project_id, default_scopes=self.AUTH_SCOPES)
            # Don't apply audience if the credentials file passed from user.
            if hasattr(credentials, "with_gdch_audience"):
                credentials = credentials.with_gdch_audience(api_audience if api_audience else host)

        # If the credentials are service account credentials, then always try to use self signed JWT.
        if always_use_jwt_access and isinstance(credentials, service_account.Credentials) and hasattr(service_account.Credentials, "with_always_use_jwt_access"):
            credentials = credentials.with_always_use_jwt_access(True)

        # Save the credentials.
        self._credentials = credentials

        # Save the hostname. Default to port 443 (HTTPS) if none is specified.
        if ':' not in host:
            host += ':443'
        self._host = host

        self._wrapped_methods: Dict[Callable, Callable] = {}

    @property
    def host(self):
        return self._host

    def _prep_wrapped_messages(self, client_info):
        # Precompute the wrapped methods.
        self._wrapped_methods = {
         }

    def close(self):
        """Closes resources associated with the transport.

       .. warning::
            Only call this method if the transport is NOT shared
            with other clients - this may cause errors in other clients!
        """
        raise NotImplementedError()

    @property
    def kind(self) -> str:
        raise NotImplementedError()


__all__ = (
    'SecretManagerServiceTransport',
)
`
	if diff := cmp.Diff(wantBase, string(baseBytes)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	// Verify services/secret_manager_service/transports/grpc.py
	grpcFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "grpc.py")
	grpcBytes, err := os.ReadFile(grpcFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grpcBytes), "class SecretManagerServiceGrpcTransport(") {
		t.Errorf("expected %s to contain class SecretManagerServiceGrpcTransport", grpcFile)
	}

	// Verify services/secret_manager_service/transports/grpc_asyncio.py
	grpcAsyncFile := filepath.Join(outdir, "google", "cloud", "secretmanager_v1", "services", "secret_manager_service", "transports", "grpc_asyncio.py")
	grpcAsyncBytes, err := os.ReadFile(grpcAsyncFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grpcAsyncBytes), "class SecretManagerServiceGrpcAsyncIOTransport(") {
		t.Errorf("expected %s to contain class SecretManagerServiceGrpcAsyncIOTransport", grpcAsyncFile)
	}
}

func TestValidateOutputContainment(t *testing.T) {
	outdir := t.TempDir()
	files := []language.GeneratedFile{
		{OutputPath: "google/cloud/secretmanager_v1/gapic_version.py"},
		{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
	}
	if err := validateOutputContainment(outdir, files); err != nil {
		t.Errorf("validateOutputContainment() error = %v, want nil", err)
	}
}

func TestValidateOutputContainment_Error(t *testing.T) {
	outdir := t.TempDir()

	for _, test := range []struct {
		name    string
		files   []language.GeneratedFile
		wantErr error
	}{
		{
			name: "empty output path",
			files: []language.GeneratedFile{
				{OutputPath: ""},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "absolute output path",
			files: []language.GeneratedFile{
				{OutputPath: filepath.Join(outdir, "escaped.py")},
			},
			wantErr: ErrInvalidOutputPath,
		},
		{
			name: "file escaping outdir via ..",
			files: []language.GeneratedFile{
				{OutputPath: "../escaped.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "file escaping outdir deeply",
			files: []language.GeneratedFile{
				{OutputPath: "google/../../../escaped.py"},
			},
			wantErr: ErrEscapeOutputDir,
		},
		{
			name: "duplicate output path",
			files: []language.GeneratedFile{
				{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
				{OutputPath: "google/cloud/secretmanager_v1/py.typed"},
			},
			wantErr: ErrDuplicateOutputPath,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateOutputContainment(outdir, test.files)
			if !errors.Is(err, test.wantErr) {
				t.Errorf("validateOutputContainment() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
