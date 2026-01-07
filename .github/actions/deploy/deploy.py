#!/usr/bin/env python3
"""Enhanced Data Science Pipelines Deployment Script.

This script provides flexible deployment options for:
1. Data Science Pipelines Operator (DSPO)
2. PyPI Server for Python packages
3. Data Science Pipelines (either via DSPO CR or direct manifests)

Note: Assumes Docker images are already built and available in the registry
via the build action.
"""

import argparse
import os
import subprocess
import tempfile
from typing import Any, Dict

from deployment_manager import K8sDeploymentManager
from deployment_manager import ResourceType
from deployment_manager import WaitCondition
from tls_manager import TLSCertificateManager
import yaml


class DSPDeployer:

    def __init__(self, args):
        self.args = args
        self.repo_owner = None
        self.target_branch = None
        self.operator_repo_path = None
        self.temp_dir = None
        self.deployment_namespace = None  # Will be set based on deployment mode
        self.dspa_name = None  # DSPA resource name
        self.operator_namespace = None
        self.operator_deployment = 'data-science-pipelines-operator-controller-manager'
        self.external_db_namespace = 'test-mariadb'
        self.is_operator_deployment = None  # Track deployment mode for port forwarding

        # Convert string arguments to booleans once
        self._convert_args_to_booleans()

        # Initialize TLS certificate manager for kind clusters
        self.tls_manager = None

        # Initialize deployment manager for K8s operations
        self.deployment_manager = K8sDeploymentManager()

    def str_to_bool(self, value: str) -> bool:
        """Convert string values to boolean."""
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            return value.lower() == 'true'
        return bool(value)

    def _convert_args_to_booleans(self):
        """Convert string arguments to boolean values."""
        boolean_args = [
            'deploy_pypi_server', 'deploy_external_argo', 'proxy',
            'cache_enabled', 'multi_user', 'artifact_proxy', 'forward_port',
            'pod_to_pod_tls_enabled', 'deploy_external_db'
        ]

        for arg_name in boolean_args:
            if hasattr(self.args, arg_name):
                current_value = getattr(self.args, arg_name)
                setattr(self.args, arg_name, self.str_to_bool(current_value))

    def setup_environment(self):
        """Extract repository information and set up environment."""
        print('🔧 Setting up deployment environment...')

        # Extract repo owner from github repository
        if self.args.github_repository:
            self.repo_owner = self.args.github_repository.split('/')[0]
            print(f'📂 Detected repository owner: {self.repo_owner}')
        else:
            raise ValueError('GitHub repository not provided')

        # Set target branch
        self.target_branch = self.args.github_base_ref or 'main'
        print(f'🌳 Target branch: {self.target_branch}')

        # Determine operator namespace
        if self.repo_owner == 'red-hat-data-services':
            self.operator_namespace = 'rhods'
        else:
            self.operator_namespace = 'opendatahub'

        # Create temp directory for operations
        self.temp_dir = tempfile.mkdtemp()
        print(f'📁 Working directory: {self.temp_dir}')

        # Set deployment namespace from input args
        self.deployment_namespace = self.args.namespace
        print(f'🏷️  Deployment namespace: {self.deployment_namespace}')

        self.dspa_name = self.args.dspa_name
        if not self.dspa_name:
            self.dspa_name = 'dspa-test'
        print(f'🏷️  Deployment Name: {self.dspa_name}')

        if not self.args.deploy_external_db:
            print(f'🏷️  External DB Namespace: {self.external_db_namespace}')

        # Create deployment namespace early since it's needed for secrets
        print(f'🏷️  Creating deployment namespace: {self.deployment_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.deployment_namespace],
            check=False  # Don't fail if namespace already exists
        )

        # Initialize TLS manager for kind clusters after namespace setup
        if self.args.pod_to_pod_tls_enabled:
            self.tls_manager = TLSCertificateManager(self)
            print('🔐 TLS certificate manager initialized for podToPodTLS')

    def clone_operator_repo(self) -> str:
        """Clone data-science-pipelines-operator repository."""
        operator_repo_url = f'https://github.com/{self.repo_owner}/data-science-pipelines-operator'
        operator_path = os.path.join(self.temp_dir,
                                     'data-science-pipelines-operator')

        print(f'📥 Cloning operator repository: {operator_repo_url}')
        self.deployment_manager.run_command(
            ['git', 'clone', operator_repo_url, operator_path])

        # Map target branch to operator branch (master -> main for operator repo)
        operator_branch = 'main' if self.target_branch == 'master' else self.target_branch

        print(f'🔄 Checking out branch: {operator_branch}')
        try:
            self.deployment_manager.run_command(
                ['git', 'checkout', operator_branch], cwd=operator_path)
        except subprocess.CalledProcessError:
            print(
                f'⚠️  Branch {operator_branch} not found, using default branch')

        # Fix Makefile permissions if it exists
        makefile_path = os.path.join(operator_path, 'Makefile')
        if os.path.exists(makefile_path):
            print('🔧 Fixing Makefile permissions...')
            self.deployment_manager.run_command(['chmod', '644', makefile_path],
                                                cwd=operator_path,
                                                check=False)

        self.operator_repo_path = operator_path
        return operator_path

    def needs_operator_repo(self) -> bool:
        """Check if we need to clone the operator repository."""
        return self.args.deploy_pypi_server

    def create_operator_namespace(self):
        # Create operator namespace if it doesn't exist
        print(f'🏷️  Creating operator namespace: {self.operator_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.operator_namespace],
            check=False  # Don't fail if namespace already exists
        )

    def deploy_operator(self):
        """Deploy Data Science Pipelines Operator."""
        print('🔧 Deploying Data Science Pipelines Operator...')

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        # Determine dspo branch based on target branch
        dspo_branch = 'main' if self.target_branch == 'master' else self.target_branch

        # Determine repo based on repo owner
        repo = 'opendatahub' if self.repo_owner == 'opendatahub-io' else 'rhoai'

        operator_image = f'quay.io/{repo}/data-science-pipelines-operator:{dspo_branch}'

        print(f'🏷️  Using operator image: {operator_image}')

        # Deploy using make with specified operator image
        # Set IMAGES_DSPO environment variable that the Makefile expects
        deploy_env = {'IMAGES_DSPO': operator_image, 'IMG': operator_image}

        # Add current environment variables
        deploy_env.update(os.environ)

        print(f'🔧 Setting IMAGES_DSPO={operator_image}')
        self.deployment_manager.run_command(
            ['make', 'deploy-kind', f'IMG={operator_image}'],
            cwd=self.operator_repo_path,
            env=deploy_env)

        # Debug ConfigMap creation (like tests.sh dependency verification)
        print('🔍 Checking created ConfigMaps...')
        self.deployment_manager.run_command(
            ['kubectl', 'get', 'configmaps', '-n', self.operator_namespace],
            check=False)

        # Verify ConfigMap creation (like tests.sh wait_for_dspo_dependencies)
        print('🔧 Verifying DSPO ConfigMap creation...')
        configmap_names = [
            'data-science-pipelines-operator-dspo-config',
            'dspo-config'  # Fallback in case the name doesn't have prefix
        ]

        configmap_found = False
        for cm_name in configmap_names:
            result = self.deployment_manager.run_command([
                'kubectl', 'get', 'configmap', cm_name, '-n',
                self.operator_namespace
            ],
                                                         check=False)

            if result.returncode == 0:
                print(f'✅ Found required ConfigMap: {cm_name}')
                configmap_found = True
                break

        if not configmap_found:
            print(f'⚠️  Required ConfigMaps not found. Available ConfigMaps:')
            self.deployment_manager.run_command([
                'kubectl', 'get', 'configmaps', '-n', self.operator_namespace,
                '--no-headers', '-o', 'custom-columns=NAME:.metadata.name'
            ],
                                                check=False)

        # Wait for operator to be ready using deployment manager
        print(
            f'⏳ Waiting for operator to be ready in namespace: {self.operator_namespace}...'
        )
        wait_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.DEPLOYMENT,
            namespace=self.operator_namespace,
            condition=WaitCondition.AVAILABLE,
            timeout='300s',
            all_resources=True,
            description='Data Science Pipelines Operator deployment')

        if not wait_success:
            print(
                f'⚠️  Operator did not become ready within timeout, investigating...'
            )
            # Use deployment manager's debug capabilities
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name='data-science-pipelines-operator-controller-manager',
                resource_type=ResourceType.DEPLOYMENT,
                selector='app.kubernetes.io/name=data-science-pipelines-operator'
            )
            raise RuntimeError('Operator did not become ready within timeout')

        # Configure operator for external Argo if requested
        if self.args.deploy_external_argo:
            self._configure_operator_for_external_argo(self.operator_namespace)

        print('✅ Data Science Pipelines Operator deployed successfully')

    def install_crds(self):

        # Install CRDs first to avoid ServiceMonitor errors
        print('🔧 Installing operator CRDs...')

        # 1. Apply additional CRDs from resources directory (like tests.sh)
        print('🔧 Installing additional CRDs from resources directory...')
        additional_crds_path = os.path.join(self.operator_repo_path, '.github',
                                            'resources', 'crds')
        if os.path.exists(additional_crds_path):
            self.deployment_manager.apply_resource(
                manifest_path=additional_crds_path,
                description='Additional CRDs from resources directory')

        # 2. Apply external route CRD (OpenShift specific)
        print('🔧 Installing OpenShift route CRD...')
        route_crd_path = os.path.join(self.operator_repo_path, 'config', 'crd',
                                      'external',
                                      'route.openshift.io_routes.yaml')
        if os.path.exists(route_crd_path):
            try:
                self.deployment_manager.apply_resource(
                    manifest_path=route_crd_path,
                    description='OpenShift route CRD')
            except Exception as e:
                print(
                    f'⚠️  OpenShift route CRD apply failed (expected on kind): {e}'
                )

    def deploy_pypi_server(self):
        """Deploy PyPI server using operator repository resources and upload
        packages."""
        if not self.args.deploy_pypi_server:
            return

        if not self.operator_repo_path:
            raise ValueError('Operator repository not cloned')

        # Create namespace
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', 'test-pypiserver'], check=False)

        # Deploy PyPI server using deployment manager
        pypi_resources_path = os.path.join(self.operator_repo_path, '.github',
                                           'resources', 'pypiserver', 'base')

        success = self.deployment_manager.deploy_and_wait(
            manifest_path=pypi_resources_path,
            namespace='test-pypiserver',
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='pypi-server',
            wait_timeout='60s',
            description='PyPI server')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace='test-pypiserver',
                deployment_name='pypi-server',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('PyPI server deployment failed')

        # Apply TLS configuration to relevant namespaces
        print('🔐 Applying TLS configuration for PyPI server...')
        nginx_tls_config_path = os.path.join(self.operator_repo_path, '.github',
                                             'resources', 'pypiserver', 'base',
                                             'nginx-tls-config.yaml')

        # Apply to both PyPI server namespace and deployment namespace
        for namespace in ['test-pypiserver', self.deployment_namespace]:
            print(f'🔗 Applying TLS config to namespace: {namespace}')
            self.deployment_manager.apply_resource(
                manifest_path=nginx_tls_config_path,
                namespace=namespace,
                description=f'TLS config for {namespace}')

        # Upload Python packages automatically when PyPI server is deployed
        print('📦 Uploading Python packages to PyPI server...')
        upload_script_path = os.path.join(self.operator_repo_path, '.github',
                                          'scripts', 'python_package_upload')

        self.deployment_manager.run_command(['bash', 'package_upload_run.sh'],
                                            cwd=upload_script_path)

        print('✅ PyPI server deployed and packages uploaded successfully')

    def deploy_seaweedfs(self):
        """Deploy SeaweedFS using local manifests."""
        if self.args.storage_backend != 'seaweedfs':
            return

        # Use local SeaweedFS manifests from the repository
        seaweedfs_path = './manifests/kustomize/third-party/seaweedfs/base'

        # Verify the kustomization file exists
        if not os.path.exists(
                os.path.join(seaweedfs_path, 'kustomization.yaml')):
            raise ValueError(
                f'SeaweedFS kustomization.yaml not found at {seaweedfs_path}')

        # Deploy SeaweedFS using deployment manager
        success = self.deployment_manager.deploy_and_wait(
            manifest_path=seaweedfs_path,
            namespace=self.deployment_namespace,
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='seaweedfs',
            wait_timeout='300s',
            description='SeaweedFS')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name='seaweedfs',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('SeaweedFS deployment failed')

        # Wait for SeaweedFS init job to complete using deployment manager
        job_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.JOB,
            resource_name='init-seaweedfs',
            namespace=self.deployment_namespace,
            condition=WaitCondition.COMPLETE,
            timeout='300s',
            description='SeaweedFS init job')

        if not job_success:
            print(
                '⚠️  Init job did not complete within timeout, investigating...'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name='init-seaweedfs',
                resource_type=ResourceType.JOB,
                selector='job-name=init-seaweedfs')
            print('⚠️  Continuing without waiting for init job completion...')

        print('✅ SeaweedFS deployed successfully from local manifests')

    def deploy_cert_manager(self):
        """Deploy cert-manager for certificate management."""
        if self.args.pipeline_store != 'kubernetes' and not self.args.pod_to_pod_tls_enabled:
            print(
                f'🏷️ Skipping cert-manager deployment because pipeline_store != kubernetes and pod_to_pod_tls is disabled'
            )
            return

        cert_manager_namespace = 'cert-manager'

        # Create cert-manager namespace
        print(f'🏷️  Creating cert-manager namespace: {cert_manager_namespace}')
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', cert_manager_namespace],
            check=False  # Don't fail if namespace already exists
        )

        # Apply cert-manager manifest (without namespace to handle cluster-scoped resources)
        self.deployment_manager.apply_resource(
            manifest_path='https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml',
            description='cert-manager')

        # Wait for cert-manager pods to be ready in the cert-manager namespace
        success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.POD,
            namespace=cert_manager_namespace,
            condition=WaitCondition.READY,
            timeout='120s',
            all_resources=True,
            description='cert-manager pods')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=cert_manager_namespace,
                deployment_name='cert-manager',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('Cert-manager deployment failed')

        print('✅ Cert-manager deployed successfully')

    def apply_webhooks(self):
        """Apply webhook certificates for TLS communication."""
        if self.args.pipeline_store != 'kubernetes':
            return

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for webhook certificates')

        print('📜 Applying webhook certificates for TLS communication...')

        webhook_certs_path = os.path.join(self.operator_repo_path, '.github',
                                          'resources', 'webhook')

        if os.path.exists(webhook_certs_path):
            self.deployment_manager.apply_resource(
                manifest_path=webhook_certs_path,
                namespace=self.operator_namespace,
                kustomize=True,
                description='Webhook certificates for TLS communication')
            print('✅ Webhook certificates applied for TLS communication')
        else:
            print(
                f'⚠️  Webhook certificates path not found: {webhook_certs_path}'
            )

    def deploy_external_mariadb(self):
        """Deploy MariaDB externally to external database namespace.

        This is equivalent to deploy_mariadb method from tests.sh in the
        operator repository.
        """
        if not self.args.deploy_external_db:
            return

        print('🗄️  Deploying external MariaDB...')

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for external MariaDB deployment'
            )

        # Create external database namespace
        print(
            f'🏷️  Creating external database namespace: {self.external_db_namespace}'
        )
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.external_db_namespace],
            check=False)  # Don't fail if namespace already exists

        # Path to MariaDB resources in operator repository
        mariadb_resources_path = os.path.join(self.operator_repo_path,
                                              '.github/resources/mariadb')

        if not os.path.exists(mariadb_resources_path):
            print(
                f'⚠️  MariaDB resources directory not found: {mariadb_resources_path}'
            )
            raise ValueError(
                f'MariaDB resources directory not found: {mariadb_resources_path}'
            )

        # Deploy external MariaDB using deployment manager
        success = self.deployment_manager.deploy_and_wait(
            manifest_path=mariadb_resources_path,
            namespace=self.external_db_namespace,
            kustomize=True,
            resource_type=ResourceType.DEPLOYMENT,
            resource_name='mariadb',
            wait_timeout='300s',
            description='External MariaDB')

        if not success:
            self.deployment_manager.debug_deployment_failure(
                namespace=self.external_db_namespace,
                deployment_name='mariadb',
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('External MariaDB deployment failed')

        print('✅ External MariaDB deployed successfully')

        # Configure MariaDB for TLS if pod-to-pod TLS is enabled
        if self.args.pod_to_pod_tls_enabled and self.tls_manager:
            self.tls_manager.configure_mariadb_for_tls()

    def apply_mariadb_minio_secrets_configmaps(self):
        """Apply MariaDB and MinIO Secrets and ConfigMaps to the external
        namespace.

        This is equivalent to apply_mariadb_minio_secrets_configmaps_external_namespace
        from tests.sh in the operator repository.
        """
        if not self.args.deploy_external_db:
            return

        print(
            '🔐 Applying MariaDB and MinIO Secrets and ConfigMaps to external namespace...'
        )

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for external pre-requisites')

        # Path to the kustomize directory containing the pre-requisite resources
        external_prereqs_path = os.path.join(
            self.operator_repo_path, '.github/resources/external-pre-reqs')

        if not os.path.exists(external_prereqs_path):
            print(
                f'⚠️  External pre-requisites directory not found: {external_prereqs_path}'
            )
            return

        # Apply external pre-requisites using deployment manager
        self.deployment_manager.apply_resource(
            manifest_path=external_prereqs_path,
            namespace=self.external_db_namespace,
            kustomize=True,
            description='MariaDB and MinIO Secrets and ConfigMaps')

        print('✅ MariaDB and MinIO Secrets and ConfigMaps applied successfully')

    def deploy_argo_lite(self):
        """Deploy Argo Lite using operator repository resources."""
        print('🔧 Deploying Argo Lite...')

        if not self.operator_repo_path:
            raise ValueError(
                'Operator repository not cloned for Argo Lite deployment')

        # Path to argo-lite resources in operator repository
        argo_lite_path = os.path.join(self.operator_repo_path, '.github',
                                      'resources', 'argo-lite')

        if not os.path.exists(argo_lite_path):
            raise ValueError(
                f'Argo Lite resources directory not found: {argo_lite_path}')

        # Apply Argo Lite resources using deployment manager
        self.deployment_manager.apply_resource(
            manifest_path=argo_lite_path,
            namespace=self.operator_namespace,
            kustomize=True,
            description='Argo Lite')

        print('✅ Argo Lite deployed successfully')

    def deploy_external_argo(self):
        """Deploy Argo Workflows externally using local manifests."""
        if not self.args.deploy_external_argo:
            return

        print(
            '⚙️  Deploying Argo Workflows externally using local manifests...')

        argo_version = self.args.argo_version or 'v3.6.7'

        # Update Argo version if specified
        if argo_version:
            print(
                f'📝 NOTE: Argo version {argo_version} specified, updating Argo Workflow manifests...'
            )

            # Write version to VERSION file
            version_file = './manifests/kustomize/third-party/argo/VERSION'
            with open(version_file, 'w') as f:
                f.write(argo_version + '\n')
            print(f'📄 Written {argo_version} to {version_file}')

            # Update manifests using make
            print('🔄 Updating Argo manifests...')
            self.deployment_manager.run_command([
                'make', '-C', './manifests/kustomize/third-party/argo', 'update'
            ])
            print(f'✅ Manifests updated for Argo version {argo_version}')

        # Apply CRDs from local manifests
        print('📦 Applying Argo CRDs from local manifests...')
        crds_path = './manifests/kustomize/third-party/argo/installs/namespace/cluster-scoped'

        self.deployment_manager.apply_resource(
            manifest_path=crds_path,
            kustomize=True,
            description='Argo Workflows CRDs')

        print(
            '✅ Argo Workflows CRDs deployed successfully from local manifests')

    def _configure_operator_for_external_argo(self, operator_namespace: str):
        """Configure the deployed operator to use external Argo Workflows."""
        print('🔧 Configuring operator to use external Argo Workflows...')

        # Patch the operator deployment to set DSPO_ARGOWORKFLOWSCONTROLLERS environment variable
        # This tells the operator to not deploy its own Argo and use external one instead
        patch_json = '[{"op": "add", "path": "/spec/template/spec/containers/0/env/-", "value": {"name": "DSPO_ARGOWORKFLOWSCONTROLLERS", "value": "{\\"managementState\\": \\"Removed\\"}"}}]'

        self.deployment_manager.run_command([
            'kubectl', 'patch', 'deployment', self.operator_deployment, '-n',
            operator_namespace, '--type=json', '-p', patch_json
        ])

        # Wait for the deployment to roll out with new configuration
        print(
            '⏳ Waiting for operator to restart with external Argo configuration...'
        )
        self.deployment_manager.run_command([
            'kubectl', 'rollout', 'status',
            f'deployment/{self.operator_deployment}', '-n', operator_namespace,
            '--timeout=300s'
        ])

        print('✅ Operator configured for external Argo successfully')

    def generate_dspa_yaml(self) -> Dict[str, Any]:
        """Generate DataSciencePipelinesApplication YAML."""
        print('📄 Generating DSPA configuration...')

        # Configure API server with proper fields (not environment variables)
        api_server_config = {
            'image':
                f'{self.args.image_registry}/{self.args.image_path_prefix}apiserver:{self.args.image_tag}',
            'argoDriverImage':
                f'{self.args.image_registry}/{self.args.image_path_prefix}driver:{self.args.image_tag}',
            'argoLauncherImage':
                f'{self.args.image_registry}/{self.args.image_path_prefix}launcher:{self.args.image_tag}',
            'cacheEnabled':
                self.args.cache_enabled,
            'enableOauth':
                False  # Disable OAuth to avoid TLS proxy issues
        }

        # Add CA bundle configuration when TLS is enabled
        if self.args.pod_to_pod_tls_enabled:
            api_server_config.update({
                'cABundle': {
                    'configMapName': 'openshift-service-ca.crt',
                    'configMapKey': 'service-ca.crt'
                }
            })
            print('🔧 Added CA bundle configuration for TLS communication')

        # Add Kubernetes native mode if specified
        if self.args.pipeline_store == 'kubernetes':
            api_server_config['pipelineStore'] = 'kubernetes'
            print('🔧 Enabling Kubernetes native pipeline storage')

        # Use images from registry (built by build action)
        dspa_config = {
            'apiVersion': 'datasciencepipelinesapplications.opendatahub.io/v1',
            'kind': 'DataSciencePipelinesApplication',
            'metadata': {
                'name': self.dspa_name,
                'namespace': self.deployment_namespace
            },
            'spec': {
                'dspVersion': 'v2',
                'apiServer': api_server_config,
                'persistenceAgent': {
                    'image':
                        f'{self.args.image_registry}/{self.args.image_path_prefix}persistenceagent:{self.args.image_tag}'
                },
                'scheduledWorkflow': {
                    'image':
                        f'{self.args.image_registry}/{self.args.image_path_prefix}scheduledworkflow:{self.args.image_tag}'
                },
                'mlmd': {
                    'deploy': True,
                    'envoy': {
                        'image': 'quay.io/maistra/proxyv2-ubi8:2.5.0'
                    }
                },
                'podToPodTLS': self.args.pod_to_pod_tls_enabled
            }
        }

        # Add storage configuration
        if self.args.storage_backend == 'minio':
            dspa_config['spec']['objectStorage'] = {
                'enableExternalRoute': True,
                'minio': {
                    'deploy':
                        True,
                    'image':
                        'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
                }
            }
        else:  # seaweedfs (default)
            dspa_config['spec']['objectStorage'] = {
                'externalStorage': {
                    'host':
                        f'seaweedfs.{self.deployment_namespace}.svc.cluster.local',
                    'port':
                        '8333',
                    'bucket':
                        'mlpipeline',
                    'region':
                        'us-east-1',  # Required but not used by SeaweedFS
                    'scheme':
                        'http',
                    's3CredentialsSecret': {
                        'accessKey': 'accesskey',
                        'secretKey': 'secretkey',
                        'secretName': 'mlpipeline-minio-artifact'
                    }
                }
            }

        # Add database configuration with MariaDB image override
        if self.args.deploy_external_db:
            dspa_config['spec']['database'] = {
                'customExtraParams': '{"tls":"true"}',
                'externalDB': {
                    'host':
                        f'mariadb.{self.external_db_namespace}.svc.cluster.local',
                    'port':
                        '3306',
                    'username':
                        'mlpipeline',
                    'pipelineDBName':
                        'mlpipeline',
                    'passwordSecret': {
                        'name': 'ds-pipeline-db-test',
                        'key': 'password'
                    }
                }
            }
        else:
            dspa_config['spec']['database'] = {
                'mariaDB': {
                    'deploy': True,
                    'image': 'quay.io/sclorg/mariadb-105-c9s:latest',
                }
            }
        return dspa_config

    def deploy_dsp_via_operator(self):
        """Deploy Data Science Pipelines via operator using DSPA CR."""
        print('🚀 Deploying Data Science Pipelines via operator...')

        # Generate DSPA configuration
        dspa_config = self.generate_dspa_yaml()

        # Write DSPA to file
        dspa_file = os.path.join(self.temp_dir, 'dspa.yaml')
        with open(dspa_file, 'w') as f:
            yaml.dump(dspa_config, f, default_flow_style=False)

        print(f'📝 DSPA configuration written to: {dspa_file}')
        print(
            f'📄 DSPA Content:\n{yaml.dump(dspa_config, default_flow_style=False)}'
        )

        # Create namespace if it doesn't exist
        self.deployment_manager.run_command(
            ['kubectl', 'create', 'namespace', self.deployment_namespace],
            check=False)

        # Note: OpenShift service CA ConfigMap is created by TLS manager when pod_to_pod_tls_enabled=true

        # Create DSPA deployment using deployment manager
        print('Creating DSPA deployment')
        self.deployment_manager.apply_resource(
            manifest_path=dspa_file,
            namespace=self.deployment_namespace,
            description='DSPA (Data Science Pipelines Application)')

        deployment_name = f'ds-pipeline-{self.dspa_name}'

        # Two-stage wait pattern from test-run.sh:
        # 1. First wait for deployment to exist (operator creates it from DSPA CR)
        print(f'⏳ Step 1: Waiting for DSPA operator to create deployment...')
        exists_success = self.deployment_manager.wait_for_resource_to_exist(
            resource_type=ResourceType.DEPLOYMENT,
            resource_name=deployment_name,
            namespace=self.deployment_namespace,
            timeout_seconds=120,  # 2 minutes for operator to create deployment
            description=f'DSPA deployment {deployment_name}')

        if not exists_success:
            print('❌ DSPA operator did not create deployment within timeout')
            print(
                f'🔍 Investigating operator deployment: {self.operator_deployment}'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name=self.operator_deployment,
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError(
                'DSPA operator did not create deployment within timeout')

        # 2. Then wait for deployment to become available
        print(
            f'⏳ Step 2: Waiting for deployment {deployment_name} to be available...'
        )
        wait_success = self.deployment_manager.wait_for_resource(
            resource_type=ResourceType.DEPLOYMENT,
            resource_name=deployment_name,
            namespace=self.deployment_namespace,
            condition=WaitCondition.AVAILABLE,
            timeout='600s',  # 10 minutes for deployment to become available
            description=f'DSPA deployment {deployment_name}')

        if wait_success:
            print('✅ Data Science Pipelines deployed via operator successfully')
        else:
            print('❌ DSPA deployment did not become ready within timeout')
            # Use deployment manager's debug capabilities for both deployments
            print(f'🔍 Investigating DSPA deployment: {deployment_name}')
            self.deployment_manager.debug_deployment_failure(
                namespace=self.deployment_namespace,
                deployment_name=deployment_name,
                resource_type=ResourceType.DEPLOYMENT)
            print(
                f'🔍 Investigating operator deployment: {self.operator_deployment}'
            )
            self.deployment_manager.debug_deployment_failure(
                namespace=self.operator_namespace,
                deployment_name=self.operator_deployment,
                resource_type=ResourceType.DEPLOYMENT)
            raise RuntimeError('DSPA did not become ready within timeout')

    def deploy_dsp_direct(self):
        """Deploy Data Science Pipelines using direct manifests (existing
        logic)"""
        print('🚀 Deploying Data Science Pipelines using direct manifests...')

        # Configure deployment arguments
        deploy_args = []

        if self.args.proxy:
            deploy_args.append('--proxy')

        if not self.args.cache_enabled:
            deploy_args.append('--cache-disabled')

        if self.args.pipeline_store == 'kubernetes':
            deploy_args.append('--deploy-k8s-native')

        if self.args.multi_user:
            deploy_args.append('--multi-user')

        if self.args.artifact_proxy:
            deploy_args.append('--artifact-proxy')

        if self.args.storage_backend and self.args.storage_backend != 'seaweedfs':
            deploy_args.extend(['--storage', self.args.storage_backend])

        if self.args.argo_version:
            deploy_args.extend(['--argo-version', self.args.argo_version])

        if self.args.pod_to_pod_tls_enabled:
            deploy_args.append('--tls-enabled')

        # Set up environment with correct REGISTRY variable
        deploy_env = os.environ.copy()
        deploy_env['REGISTRY'] = self.args.image_registry

        print(f'🔧 Setting REGISTRY={self.args.image_registry}')
        print(f'🏷️  Using image tag: {self.args.image_tag}')

        # Call existing deploy script
        deploy_script = './.github/resources/scripts/deploy-kfp.sh'
        cmd = ['bash', deploy_script] + deploy_args

        # Add timeout to prevent hanging on log collection
        self.deployment_manager.run_command(
            cmd, timeout=1800, env=deploy_env)  # 30 minute timeout

        print('✅ Data Science Pipelines deployed directly successfully')

    def forward_port(self):
        """Forward API server port to localhost."""
        if not self.args.forward_port:
            return

        print('🔗 Setting up port forwarding...')

        # Use different app names based on deployment mode
        if self.is_operator_deployment:
            # Operator deployment uses ds-pipeline-{dspa_name}
            api_server_app_name = f'ds-pipeline-{self.dspa_name}'
        else:
            # Direct deployment uses ml-pipeline
            api_server_app_name = 'ml-pipeline'

        print(f'🔍 Looking for pods with app label: {api_server_app_name}')

        # First, check if any running pods exist with the expected label
        pods_result = self.deployment_manager.run_command([
            'kubectl', 'get', 'pods', '-n', self.deployment_namespace, '-l',
            f'app={api_server_app_name}',
            '--field-selector=status.phase=Running', '--no-headers', '-o',
            'custom-columns=NAME:.metadata.name'
        ],
                                                          check=False)

        if pods_result.returncode != 0 or not pods_result.stdout.strip():
            print(
                f'❌ No running pods found with label app={api_server_app_name} in namespace {self.deployment_namespace}'
            )
            print('🔍 Available pods in namespace:')
            self.deployment_manager.run_command([
                'kubectl', 'get', 'pods', '-n', self.deployment_namespace, '-o',
                'wide'
            ],
                                                check=False)
            raise RuntimeError(
                f'Port forwarding failed: No running API server pods found with label app={api_server_app_name}'
            )

        pod_names = [
            name.strip()
            for name in pods_result.stdout.strip().split('\n')
            if name.strip()
        ]
        print(
            f'✅ Found {len(pod_names)} running pod(s): {", ".join(pod_names)}')

        # Attempt port forwarding
        forward_script = './.github/resources/scripts/forward-port.sh'
        try:
            self.deployment_manager.run_command([
                'bash', forward_script, '-q', self.deployment_namespace,
                api_server_app_name, '8888', '8888'
            ])
            print('✅ Port forwarding setup completed')
        except subprocess.CalledProcessError as e:
            print(f'❌ Port forwarding failed with exit code {e.returncode}')
            if e.output:
                print(f'❌ Error output: {e.output}')
            raise RuntimeError(f'Port forwarding setup failed: {str(e)}')

    def deploy(self):
        """Main deployment orchestration with intelligent mode selection."""
        try:
            self.setup_environment()

            # 🧠 Intelligent deployment mode selection
            use_operator_deployment = self._should_use_operator_deployment()
            self.is_operator_deployment = use_operator_deployment

            if use_operator_deployment:
                print('🔧 Using DSPO (operator) deployment mode')
                # For operator deployment, we always need the operator repo
                self.clone_operator_repo()

                # Create Operator Namespace
                self.create_operator_namespace()

                # Install CRDs
                self.install_crds()

                # Deploy cert-manager (required for TLS certificates on kind)
                self.deploy_cert_manager()

                # Set up TLS certificates for kind cluster (after cert-manager)
                if self.args.pod_to_pod_tls_enabled and self.tls_manager:
                    self.tls_manager.setup_tls_for_kind()

                # Deploy Argo Lite
                self.deploy_argo_lite()

                # Deploy external Argo if requested (must be done before operator)
                self.deploy_external_argo()

                # Deploy operator
                self.deploy_operator()

                # Apply webhook certificates for TLS communication
                self.apply_webhooks()

                # Deploy PyPI server if requested (includes package upload)
                self.deploy_pypi_server()

                # Apply MariaDB and MinIO secrets and configmaps before deploying SeaweedFS
                self.apply_mariadb_minio_secrets_configmaps()

                # Deploy external MariaDB if requested
                self.deploy_external_mariadb()

                # Deploy SeaweedFS if using seaweedfs storage (like tests.sh approach)
                self.deploy_seaweedfs()

                # Deploy DSP via operator
                self.deploy_dsp_via_operator()

            else:
                print(
                    '🔧 Using direct manifest deployment mode (multi-user detected)'
                )

                # Check if we need operator repo for PyPI server features only
                if self.args.deploy_pypi_server:
                    self.clone_operator_repo()
                    self.deploy_pypi_server()

                # Deploy DSP directly
                self.deploy_dsp_direct()

            # Setup port forwarding
            self.forward_port()

            print('🎉 Deployment completed successfully!')

        except Exception as e:
            print(f'❌ Deployment failed: {str(e)}')
            raise
        finally:
            # Cleanup temp directory
            if self.temp_dir and os.path.exists(self.temp_dir):
                import shutil
                shutil.rmtree(self.temp_dir)

    def _should_use_operator_deployment(self) -> bool:
        """Determine whether to use DSPO (operator) or direct deployment.

        Logic:
        - Multi-user mode: DSPO doesn't support it → use direct deployment
        - All other cases: Use DSPO deployment (default)
        """
        if self.args.multi_user:
            print(
                "⚠️  Multi-user mode detected: DSPO doesn't support multi-user, using direct deployment"
            )
            return False
        elif self.args.proxy:
            print(
                "⚠️  Proxy mode detected: DSPO doesn't support proxy, using direct deployment"
            )
            return False

        return True

    def _create_mariadb_certificate(
            self, base_cert: Dict[str, Any]) -> Dict[str, Any]:
        """Create MariaDB TLS certificate based on the main certificate.

        Args:
            base_cert: The base certificate dictionary to clone

        Returns:
            Dictionary containing the MariaDB certificate manifest
        """
        # Expected secret name format: ds-pipelines-mariadb-tls-{dspa-name}
        mariadb_secret_name = f'ds-pipelines-mariadb-tls-{self.dspa_name}'
        mariadb_cert_name = f'mariadb-tls-cert-{self.dspa_name}'

        print(
            f'🔧 Creating MariaDB certificate with secret name: {mariadb_secret_name}'
        )

        # Clone the base certificate
        mariadb_cert = yaml.safe_load(yaml.dump(base_cert))

        # Update metadata
        mariadb_cert['metadata']['name'] = mariadb_cert_name

        # Update spec for MariaDB-specific settings
        mariadb_cert['spec']['commonName'] = f'ds-pipeline-db-{self.dspa_name}'
        mariadb_cert['spec']['secretName'] = mariadb_secret_name

        # Update DNS names for MariaDB service
        mariadb_cert['spec']['dnsNames'] = [
            f'ds-pipeline-db-{self.dspa_name}',
            f'ds-pipeline-db-{self.dspa_name}.{self.deployment_namespace}',
            f'ds-pipeline-db-{self.dspa_name}.{self.deployment_namespace}.svc.cluster.local',
            'localhost'
        ]

        return mariadb_cert


def main():
    parser = argparse.ArgumentParser(
        description='Deploy Data Science Pipelines')

    # GitHub context
    parser.add_argument(
        '--github-repository',
        required=True,
        help='GitHub repository (owner/repo)')
    parser.add_argument(
        '--github-base-ref', help='GitHub base ref (target branch)')

    # Image configuration (images already built by build action)
    parser.add_argument('--image-tag', required=True, help='Image tag')
    parser.add_argument(
        '--image-registry', required=True, help='Image registry')
    parser.add_argument(
        '--image-path-prefix',
        required=False,
        default='',
        help='Image path prefix to add')

    # PyPI deployment options (consolidated)
    parser.add_argument(
        '--deploy-pypi-server',
        default='false',
        help='Deploy PyPI server and upload packages')
    parser.add_argument(
        '--deploy-external-argo',
        default='false',
        help='Deploy Argo Workflows externally in separate namespace')

    # Existing KFP options
    parser.add_argument(
        '--pipeline-store',
        default='database',
        choices=['database', 'kubernetes'],
        help='Pipeline store type')
    parser.add_argument('--proxy', default='false', help='Enable proxy')
    parser.add_argument('--cache-enabled', default='true', help='Enable cache')
    parser.add_argument('--multi-user', default='false', help='Multi-user mode')
    parser.add_argument(
        '--artifact-proxy', default='false', help='Enable artifact proxy')
    parser.add_argument(
        '--storage-backend',
        default='seaweedfs',
        choices=['seaweedfs', 'minio'],
        help='Storage backend')
    parser.add_argument('--argo-version', help='Argo version')
    parser.add_argument(
        '--forward-port', default='true', help='Forward API server port')
    parser.add_argument(
        '--pod-to-pod-tls-enabled',
        default='false',
        help='Enable pod-to-pod TLS')
    parser.add_argument(
        '--namespace', default='kubeflow', help='Namespace for DSPA deployment')

    parser.add_argument(
        '--deploy-external-db',
        default=False,
        help='If you want to deploy DB externally and not via DSPO')

    parser.add_argument(
        '--dspa-name', default='dspa-test', help='Name of your dspa name')

    args = parser.parse_args()

    deployer = DSPDeployer(args)
    deployer.deploy()


if __name__ == '__main__':
    main()
