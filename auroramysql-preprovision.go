package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/nu7hatch/gouuid"
	"github.com/robfig/cron"
)

type DBParams struct {
	Dbname                  string
	Instanceid              string
	Masterusername          string
	Masterpassword          string
	Securitygroupid         string
	Allocatedstorage        int64
	Autominorversionupgrade bool
	Dbinstanceclass         string
	Dbparametergroupname    string
	Dbsubnetgroupname       string
	Multiaz                 bool
	Storageencrypted        bool
	Endpoint                string
	ReaderEndpoint          string
}

func run() {

	setupDB()

	fmt.Println("Checking for provisions")
	provision_micro, _ := strconv.Atoi(os.Getenv("PROVISION_MICRO"))
	provision_small, _ := strconv.Atoi(os.Getenv("PROVISION_SMALL"))
	provision_medium, _ := strconv.Atoi(os.Getenv("PROVISION_MEDIUM"))
	provision_large, _ := strconv.Atoi(os.Getenv("PROVISION_LARGE"))

	if need("micro", provision_micro) {
		record(provision_hobby("micro"), "micro")
	}

	if need("small", provision_small) {
		record(provision_hobby("small"), "small")
	}

	if need("medium", provision_medium) {
		record(provision_hobby("medium"), "medium")
	}

	if need("large", provision_large) {
		record(provision("large"), "large")
	}
}

func main() {
	if os.Getenv("REGION") == "" {
		fmt.Println("REGION was not specified.")
		os.Exit(2)
	}
	if os.Getenv("HOBBY_DB") == "" {
		fmt.Println("HOBBY_DB was not specified.")
		os.Exit(2)
	}
	if os.Getenv("BROKER_DB") == "" {
		fmt.Println("BROKER_DB was not specified.")
		os.Exit(2)
	}
	if strings.Index(os.Getenv("HOBBY_DB"), "@") == -1 {
		fmt.Println("HOBBY_DB was not a valid mysql db uri.  E.g., user:pass@tcp(host:3306)/db")
		os.Exit(2)
	}
	if os.Getenv("ENVIRONMENT") == "" || strings.Index(os.Getenv("ENVIRONMENT"), "-") > -1 || strings.Index(os.Getenv("ENVIRONMENT"), "_") > -1 {
		fmt.Println("ENVIORNMENT was not set or had an invalid character, it can only be alpha numeric.")
		os.Exit(2)
	}
	if os.Getenv("RUN_AS_CRON") != "" {
		c := cron.New()
		c.AddFunc("@every 1m", run)
		c.Run()
	} else {
		run()
	}
}

func setupDB() {
	// Initialize Database
	uri := os.Getenv("BROKER_DB")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer db.Close()

	initstatement := `CREATE TABLE if not exists aurora_mysql_provision (
			name character varying(200),
			plan character varying(200),
			claimed character varying(200),
			make_date timestamp with time zone,
			masterpass character varying(200),
			masteruser character varying(200),
			endpoint character varying(200),
			reader_endpoint character varying(200)
		);`

	_, err = db.Exec(initstatement)
	if err != nil {
		log.Fatal("Unable to create database: %s\n", err)
	}

}

func record(dbparams DBParams, plan string) {
	uri := os.Getenv("BROKER_DB")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer db.Close()
	var newname string
	err = db.QueryRow("INSERT INTO aurora_mysql_provision(name,plan,claimed,make_date,masterpass,masteruser,endpoint,reader_endpoint) VALUES($1,$2,$3,now(),$4,$5,$6,$7) returning name;", dbparams.Dbname, plan, "no", dbparams.Masterpassword, dbparams.Masterusername, dbparams.Endpoint, dbparams.ReaderEndpoint).Scan(&newname)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println("Recorded new sql database:", newname)

}

func provision_hobby(plan string) DBParams {
	uri := os.Getenv("HOBBY_DB")
	dbparams := new(DBParams)

	//Generate Unique DB Name
	dbnameuuid, _ := uuid.NewV4()
	dbparams.Dbname = os.Getenv("ENVIRONMENT") + "ma" + strings.Split(dbnameuuid.String(), "-")[0]
	fmt.Println("Provision: Hobby DB Name: " + dbparams.Dbname)
	dbparams.Instanceid = dbparams.Dbname

	//Generate Unique User ID
	usernameuuid, _ := uuid.NewV4()
	dbparams.Masterusername = "u" + strings.Split(usernameuuid.String(), "-")[0]
	fmt.Println("Provision: Hobby User Name: " + dbparams.Masterusername)

	//Generate Unique Password
	passworduuid, _ := uuid.NewV4()
	dbparams.Masterpassword = strings.Split(passworduuid.String(), "-")[0] + strings.Split(passworduuid.String(), "-")[1]
	fmt.Println("Provision: Hobby Password: " + dbparams.Masterpassword)

	db, err := sql.Open("mysql", uri)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer db.Close()

	_, dberr := db.Exec("CREATE USER '" + dbparams.Masterusername + "' IDENTIFIED BY '" + dbparams.Masterpassword + "'")
	fmt.Println("Provision: creating user (hobby)")
	if dberr != nil {
		fmt.Println(dberr)
		os.Exit(2)
	}
	//dbparams.Masterusername
	_, dberr = db.Exec("CREATE DATABASE " + dbparams.Dbname)
	fmt.Println("Provision: granting permission (hobby)")
	if dberr != nil {
		fmt.Println(dberr)
		os.Exit(2)
	}

	_, dberr = db.Exec("GRANT ALL ON " + dbparams.Dbname + ".*  TO '" + dbparams.Masterusername + "'")
	fmt.Println("Provision: granting permission (hobby)")
	if dberr != nil {
		fmt.Println(dberr)
		os.Exit(2)
	}

	dbparams.Endpoint = strings.Replace(strings.Split(os.Getenv("HOBBY_DB"), "@")[1], "/aurora_mysql_db", "", 1) + "/" + dbparams.Dbname
	dbparams.ReaderEndpoint = dbparams.Endpoint
	return *dbparams
}

func provision(plan string) DBParams {
	region := os.Getenv("REGION")
	dbparams := new(DBParams)

	dbnameuuid, _ := uuid.NewV4()
	dbparams.Dbname = os.Getenv("ENVIRONMENT") + "ma" + strings.Split(dbnameuuid.String(), "-")[0]
	fmt.Println("Provision: database name: ", dbparams.Dbname)
	dbparams.Instanceid = dbparams.Dbname

	usernameuuid, _ := uuid.NewV4()
	dbparams.Masterusername = "u" + strings.Split(usernameuuid.String(), "-")[0]
	fmt.Println("Provision: username: ", dbparams.Masterusername)

	passworduuid, _ := uuid.NewV4()
	dbparams.Masterpassword = strings.Split(passworduuid.String(), "-")[0] + strings.Split(passworduuid.String(), "-")[1]
	fmt.Println("Provision: password: ", dbparams.Masterpassword)

	dbparams.Securitygroupid = os.Getenv("RDS_SECURITY_GROUP")

	switch plan {
	case "small":
		dbparams.Autominorversionupgrade = true
		dbparams.Dbinstanceclass = os.Getenv("SMALL_INSTANCE_TYPE")
		dbparams.Dbparametergroupname = "rds-auroramysql-small-" + os.Getenv("ENVIRONMENT")
		dbparams.Dbsubnetgroupname = "rds-auroramysql-subnet-group-" + os.Getenv("ENVIRONMENT")
		dbparams.Multiaz = false
		dbparams.Storageencrypted = false
	case "medium":
		dbparams.Autominorversionupgrade = false
		dbparams.Dbinstanceclass = os.Getenv("MEDIUM_INSTANCE_TYPE")
		dbparams.Dbparametergroupname = "rds-auroramysql-medium-" + os.Getenv("ENVIRONMENT")
		dbparams.Dbsubnetgroupname = "rds-auroramysql-subnet-group-" + os.Getenv("ENVIRONMENT")
		dbparams.Multiaz = false
		dbparams.Storageencrypted = false
	case "large":
		dbparams.Autominorversionupgrade = false
		dbparams.Dbinstanceclass = os.Getenv("LARGE_INSTANCE_TYPE")
		dbparams.Dbparametergroupname = "rds-auroramysql-large-" + os.Getenv("ENVIRONMENT")
		dbparams.Dbsubnetgroupname = "rds-auroramysql-subnet-group-" + os.Getenv("ENVIRONMENT")
		dbparams.Multiaz = true
		dbparams.Storageencrypted = true
	}
	svc := rds.New(session.New(&aws.Config{
		Region: aws.String(region),
	}))

	params := &rds.CreateDBClusterInput{
		DBClusterIdentifier:         aws.String(dbparams.Instanceid), // Required
		Engine:                      aws.String("aurora"),            // Required
		MasterUsername:              aws.String(dbparams.Masterusername),
		MasterUserPassword:          aws.String(dbparams.Masterpassword),
		DBSubnetGroupName:           aws.String(dbparams.Dbsubnetgroupname),
		Port:                        aws.Int64(3306),
		DatabaseName:                aws.String(dbparams.Dbname),
		DBClusterParameterGroupName: aws.String("rds-auroramysql-cluster-" + os.Getenv("ENVIRONMENT")),
		StorageEncrypted:            aws.Bool(dbparams.Storageencrypted),
		Tags: []*rds.Tag{
			{ // Required
				Key:   aws.String("Name"),
				Value: aws.String(dbparams.Dbname),
			},
			{ // Required
				Key:   aws.String("billingcode"),
				Value: aws.String("pre-provisioned"),
			},
		},
		VpcSecurityGroupIds: []*string{
			aws.String(dbparams.Securitygroupid), // Required
		},
	}
	response, err := svc.CreateDBCluster(params)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	fmt.Println("Provision: Created db cluster with endpoint ", *response.DBCluster.Endpoint)
	dbparams.Endpoint = "tcp(" + *response.DBCluster.Endpoint + ":3306)/" + dbparams.Dbname
	dbparams.ReaderEndpoint = "tcp(" + *response.DBCluster.ReaderEndpoint + ":3306)/" + dbparams.Dbname

	params_instance := &rds.CreateDBInstanceInput{
		DBInstanceClass:         aws.String(dbparams.Dbinstanceclass),
		DBClusterIdentifier:     aws.String(dbparams.Instanceid),                      // Required
		DBInstanceIdentifier:    aws.String(dbparams.Instanceid + "-" + region + "a"), // Required
		Engine:                  aws.String("aurora"),                                 // Required
		AutoMinorVersionUpgrade: aws.Bool(dbparams.Autominorversionupgrade),
		DBSubnetGroupName:       aws.String(dbparams.Dbsubnetgroupname),
		StorageEncrypted:        aws.Bool(dbparams.Storageencrypted),
		Tags: []*rds.Tag{
			{ // Required
				Key:   aws.String("Name"),
				Value: aws.String(dbparams.Dbname),
			},
			{ // Required
				Key:   aws.String("billingcode"),
				Value: aws.String("pre-provisioned"),
			},
		},
	}

	_, err2 := svc.CreateDBInstance(params_instance)
	if err2 != nil {
		fmt.Println(err2.Error())
		os.Exit(2)
	}
	fmt.Println("Provision: Create rw instance in db cluster")

	if dbparams.Multiaz == true {
		params_read_inst := &rds.CreateDBInstanceInput{
			DBInstanceClass:         aws.String(dbparams.Dbinstanceclass),
			DBClusterIdentifier:     aws.String(dbparams.Instanceid),                      // Required
			DBInstanceIdentifier:    aws.String(dbparams.Instanceid + "-" + region + "b"), // Required
			Engine:                  aws.String("aurora"),                                 // Required
			AutoMinorVersionUpgrade: aws.Bool(dbparams.Autominorversionupgrade),
			DBSubnetGroupName:       aws.String(dbparams.Dbsubnetgroupname),
			StorageEncrypted:        aws.Bool(dbparams.Storageencrypted),
			Tags: []*rds.Tag{
				{ // Required
					Key:   aws.String("Name"),
					Value: aws.String(dbparams.Dbname),
				},
				{ // Required
					Key:   aws.String("billingcode"),
					Value: aws.String("pre-provisioned"),
				},
			},
		}

		_, err3 := svc.CreateDBInstance(params_read_inst)
		if err3 != nil {
			fmt.Println(err3.Error())
			os.Exit(2)
		}
		fmt.Println("Provision: Create read only instance in db cluster")
	}

	return *dbparams
}

func need(plan string, minimum int) bool {
	uri := os.Getenv("BROKER_DB")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer db.Close()

	var unclaimedcount int
	err = db.QueryRow("SELECT count(*) as unclaimedcount from aurora_mysql_provision where plan='" + plan + "' and claimed='no'").Scan(&unclaimedcount)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println("Checking " + plan + " with unclaimed count " + strconv.Itoa(unclaimedcount) + " and minimum " + strconv.Itoa(minimum))
	if unclaimedcount < minimum {
		return true
	}
	return false
}
