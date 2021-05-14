This directory contains a sample Terraform configuration for a status page with a single monitor.  

## Usage

```shell script
git clone https://github.com/altinity/terraform-provider-betteruptime && \
  cd terraform-provider-betteruptime/examples/basic

echo '# See variables.tf for more.
betteruptime_api_token             = "XXXXXXXXXXXXXXXXXXXXXXXX"
betteruptime_status_page_subdomain = "example"
' > terraform.tfvars

terraform apply

# open https://${betteruptime_status_page_subdomain}.betteruptime.com  
open $(terraform output -raw betteruptime_status_page_url)
```