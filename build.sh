#!/bin/sh

cd /app
go get  "github.com/lib/pq"
go get  "github.com/aws/aws-sdk-go/aws"
go get  "github.com/aws/aws-sdk-go/aws/session"
go get  "github.com/aws/aws-sdk-go/service/rds"
go get  "github.com/nu7hatch/gouuid"
go get  "github.com/robfig/cron"
go get  "github.com/go-sql-driver/mysql"
go build auroramysql-preprovision.go

