---
page_title: "Provider: MariaDB SkySQL Terraform Provider"

description: |-
The MariaDB SkySQL Terraform Provider allows database services in MariaDB SkySQL to be managed using Terraform.

---

# SKYSQL-BETA Provider

The current implementation is in Technical preview and will become available in the Terraform registry in the near future.
It allows configuring any SkySQL DB topology using the Terraform's declarative language.

* It automatically provisions new SkySQL services or updates existing ones when the Terraform configuration is applied

* It automatically tears down SkySQL services when the Terraform configuration is destroyed

[Terraform](https://www.terraform.io/) is an open source infrastructure-as-code (IaC) utility.

Alternatively, SkySQL services can be managed interactively the [SkySQL Portal](https://skysql.mariadb.com/dashboard) or the SkySQL REST API.

Use the navigation to the left to read about the available resources.

## Installing the Terraform Provider for SkySQL

### Automated Installation (Recommended)

The Terraform Provider for SkySQL **WILL BE** listed on the [Terraform Registry](https://registry.terraform.io/providers/mariadb-corporation/skysql-beta/).

#### Create a Terraform configuration file

Providers listed on the Terraform Registry can be automatically downloaded when initializing a working directory with `terraform init`. The Terraform configuration block is used to configure some behaviors of Terraform itself, such as the Terraform version and the required providers and versions.

**Example**: A Terraform configuration block.

```hcl
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
    }
  }
}
```

You can use `version` locking and operators to require specific versions of the provider.

**Example**: A Terraform configuration block with the provider versions.

```hcl
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
       version = ">= x.y.z"
    }
  }
}
```

### Manual Installation

The latest release of the provider can be found on [`terraform-provider-skysql-beta/releases`](https://github.com/mariadb-corporation/terraform-provider-skysql-beta/releases). You can download the appropriate version of the provider for your operating system using a command line shell or a browser.

This can be useful in environments that do not allow direct access to the Internet.

#### Linux

The following examples use Bash on Linux (x64).

1. On a Linux operating system with Internet access, download the plugin from GitHub using the shell.

    ```console
    RELEASE=x.y.z
    OS=linux
    ARCH=amd64
    wget -q https://github.com/mariadb-corporation/terraform-provider-skysql-beta/releases/download/${RELEASE}/terraform-provider-skysql-beta_${RELEASE}_{OS}_{ARCH}.zip
    ```

2. Create a directory for the provider.

    > **Note**
    >
    > The directory hierarchy that Terraform uses to precisely determine the source of each provider it finds locally.
    >
    > `<registry>/<namespace>/<service>/<version>/<OS_arch>/`

    ```console
    mkdir -p ~/.terraform.d/plugins/registry.terraform.io/mariadb-corporation/skysql-beta
    ```

3. Copy the plugin to a target system and move to the Terraform plugins directory.

    ```console
    mv terraform-provider-skysql-beta_${RELEASE}_${OS}_${ARCH}.zip ~/.terraform.d/plugins/registry.terraform.io/mariadb-corporation/skysql-beta

    ```

4. Verify the presence of the plugin in the Terraform plugins directory.

    ```console
    ls ~/.terraform.d/plugins/local/mariadb-corporation/skysql-beta/
    ```

#### macOS

The following example uses Bash (default) on macOS (ARM).

1. On a macOS operating system with Internet access, install wget with [Homebrew](https://brew.sh).

    ```console
    brew install wget
    ```

2. Download the plugin from GitHub using the shell. Replace 'x.y.z' below with the release version (see above link for available releases).

    ```console
    export RELEASE=x.y.z
    wget -q https://github.com/mariadb-corporation/terraform-provider-skysql-beta/releases/download/v${RELEASE}/terraform-provider-skysql-beta_${RELEASE}_darwin_arm64.zip
    ```

3. Create a directory for the provider.

    > **Note**
    >
    > The directory hierarchy that Terraform uses to precisely determine the source of each provider it finds locally.
    >
    > `<registry>/<namespace>/<service>/<version>/<OS_arch>/`

    ```console
    mkdir -p ~/.terraform.d/plugins/registry.terraform.io/mariadb-corporation/skysql-beta/
    ```

4. Copy the plugin to a target system and move to the Terraform plugins directory.

    ```console
    mv terraform-provider-skysql-beta_${RELEASE}_darwin_arm64.zip ~/.terraform.d/plugins/registry.terraform.io/mariadb-corporation/skysql-beta/
    ```

5. Verify the presence of the plugin in the Terraform plugins directory.

    ```console
    ls ~/.terraform.d/plugins/local/mariadb-corporation/skysql-beta/
    ```

## Configure the Terraform Configuration Files

In order to use the `skysql-beta` terraform provider, you need to configure the provider in your Terraform configuration files.

```hcl
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
    }
  }
}
```

or you can specify the provider version.

```hcl
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
      version = ">= x.y.z"
    }
  }
}
```

## Verify the Terraform Initialization

To verify the initialization, navigate to the working directory for your Terraform configuration and run `terraform init`. You should see a message indicating that Terraform has been successfully initialized and the installed version of the Terraform Provider for vSphere.

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding latest version of mariadb-corporation/skysql-beta...
- Installing mariadb-corporation/skysql-beta x.y.z...
- Installed mariadb-corporation/skysql-beta x.y.z (unauthenticated)
...

Terraform has been successfully initialized!
```

## Get the Provider Version

To find the provider version, navigate to the working directory of your Terraform configuration and run `terraform version`. You should see a message indicating the provider version.

**Example**: Terraform Provider Version from the Terraform Registry

```console
$ terraform version
Terraform x.y.z
on darwin_arm64
+ provider registry.terraform.io/mariadb-corporation/skysql-beta x.y.z
```

## Create a new SkySQL service

1. Go to MariaDB ID: [https://id.mariadb.com/account/api/](https://id.mariadb.com/account/api/) and generate an API key

2. Set environment variables:

    ```bash
        export TF_SKYSQL_API_ACCESS_TOKEN=[SKYSQL API access token]
        export TF_SKYSQL_API_BASE_URL=https://api.mariadb.com
    ```

3. Create a new SkySQL service using the example below:

```terraform
terraform {
  required_providers {
    skysql = {
      source = "registry.terraform.io/mariadb-corporation/skysql-beta"
    }
  }
}

provider "skysql" {}

# Retrieve the list of available versions for each topology like standalone, masterslave, xpand-direct etc
data "skysql_versions" "default" {}


# Filter the list of versions to only include  versions for the standalone topology
locals {
  sky_versions_filtered = [
    for item in data.skysql_versions.default.versions : item if item.topology == "standalone"
  ]
}

# Retrieve the list of projects. Project is a way of grouping the services.
# Note: Next release will make project_id optional in the create service api
data "skysql_projects" "default" {}

output "skysql_projects" {
  value = data.skysql_projects.default
}

# Create a service
resource "skysql_service" "default" {
  service_type   = "transactional"
  topology       = "standalone"
  cloud_provider = "gcp"
  region         = "us-central1"
  name           = "my-first-service"
  architecture   = "amd64"
  nodes          = 1
  size           = "sky-2x8"
  storage        = 100
  ssl_enabled    = true
  version        = local.sky_versions_filtered[0].name
  # [Optional] Below you can find example with optional parameters how to configure a privatelink connection
  endpoint_mechanism        = "privatelink"
  endpoint_allowed_accounts = ["gcp-project-id"]
  # [/Optional]
  # The service create is an asynchronous operation.
  # if you want to wait for the service to be created set wait_for_creation to true
  wait_for_creation = true
}

# Retrieve the service default credentials.
# When the service is created please change the default credentials
data "skysql_credentials" "default" {
  service_id = skysql_service.default.id
}

# Retrieve the service details
data "skysql_service" "default" {
  service_id = skysql_service.default.id
}

# Show the service details
output "skysql_service" {
  value = data.skysql_service.default
}

# Show the service credentials
output "skysql_credentials" {
  value     = data.skysql_credentials.default
  sensitive = true
}

# You need to add your ip address in the CIRD format to allow list in order to connect to the service
# Note: the operation is asynchronous by default.
# If you want to wait for the operation to complete set wait_for_creation to true
resource "skysql_allow_list" "default" {
  service_id = skysql_service.default.id
  allow_list = [
    {
      "ip" : "104.28.203.45/32",
      "comment" : "homeoffice"
    }
  ]
  wait_for_creation = true
}

# Example how you can generate a command line for the database connection
output "skysql_cmd" {
  value = "mariadb --host ${data.skysql_service.default.fqdn} --port 3306 --user ${data.skysql_service.default.service_id} -p --ssl-ca ~/Downloads/skysql_chain_2022.pem"
}
```

## Limitations

* The terraform resource `skysql_service` doesn't support updates. If you need to change the configuration of a service, you need to destroy the service and create a new one.

## Secrets and Terraform state

Some resources that can be created with this provider, like `skysql_credentials`, are
considered "secrets", and as such are marked by this provider as _sensitive_, so to
help practitioner to not accidentally leak their value in logs or other form of output.

It's important to remember that the values that constitute the "state" of those
resources will be stored in the [Terraform state](https://www.terraform.io/language/state) file.
This includes the "secrets", that will be part of the state file **unencrypted**.

Because of these limitations, **use of these resources for production deployments is _not_ recommended**.
Failing that, **protecting the content of the state file is strongly recommended**.

The more general advice is that it's better to generate "secrets" outside of Terraform,
and then distribute them securely to the system where Terraform will make use of them.
