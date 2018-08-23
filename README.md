## Synopsis

Docker image which runs cronjob for provisioning of aurora mysql instances on AWS RDS

## Details
The cronjob makes sure that there are always enough unclaimed micro, small, medium, and large instances to match the provided environment variables

## Dependencies

1. "fmt"
2. "strings"
3. "database/sql"
4. "github.com/lib/pq"
5. "github.com/aws/aws-sdk-go/aws"
6. "github.com/aws/aws-sdk-go/aws/session"
7. "github.com/aws/aws-sdk-go/service/rds"
8. "github.com/nu7hatch/gouuid"
9. "os"

## Requirements
* go
* aws creds

## Runtime Environment Variables

__Database Variables__

* BROKER_DB

  URL of the Postgres database that stores information about currently provisioned instances, e.g. `postgres://user:pass@host:5432/db`

* HOBBY_DB

  URL of the shared tenancy Aurora MySQL instance for micro/hobby instances, e.g. `user:pass@tcp(host:3306)/db`

__Instance Type Variables__
  
AWS instance type to provision for a given instance, e.g. `db.t2.medium`

* SMALL_INSTANCE_TYPE
* MEDIUM_INSTANCE_TYPE
* LARGE_INSTANCE_TYPE

__Provision Variables__

Number of required unclaimed instances, e.g. `1`

* PROVISION_MICRO
* PROVISION_SMALL
* PROVISION_MEDIUM
* PROVISION_LARGE

__Other__

* RDS_SECURITY_GROUP
* RUN_AS_CRON - 0 or 1
* ENVIRONMENT - prefix for database instance names, keeps things organized