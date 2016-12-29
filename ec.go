package main

// scale = [up,down]
// scalenum = number //number of servers to scale up or down
// tag - name of the tag to look for in the aws
// OR
// poweroff //power off all aws instances
// Wait time untill power off servers 



// GLOBAL CONST
// InstanceTypeP2Xlarge is a InstanceType enum value
//    InstanceTypeP2Xlarge = "p2.xlarge"


import (
    "fmt"
    "log"
    "math/rand"
    "time"
    "strconv"
    "strings"
    "net/http"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ec2"
)

const (
   nginxconf = "/tmp/nginx.conf"
)

const (

   INSTANCE_STATE_PENDING      = "pending"
   INSTANCE_STATE_RUNNING      = "running"
   INSTANCE_STATE_SHUTTINGD    = "shutting-down"
   INSTANCE_STATE_TERMINATED   = "terminated"
   INSTANCE_STATE_STOPPING     = "stopping"
   INSTANCE_STATE_STOPPED      = "stopped"
)


type appError struct {
         Error   error
         Message string
         Code    int
    }

func Shuffle(a []string) {
    for i := range a {
        j := rand.Intn(i + 1)
        a[i], a[j] = a[j], a[i]
    }
}



func ScaleInstances(sess *session.Session, instIds []string, scalenum int)  {

	if len(instIds) > 0 {

        fmt.Println("Number of instances with stopped state which i can run = ", len(instIds))

        //randomizing instances to start 
         rand.Seed(time.Now().UnixNano())
         Shuffle(instIds)

		//taking amount of instances to scale, received via scalenum query param
			InstWillStart := instIds[0:scalenum]
			for _, v := range InstWillStart {
				startinstance(sess, v)
			}

	} else {
		println("there is NO stopped instances with requested TAG")
    }
}


func listbyTag(sess *session.Session, HostNameTagFilter string, InstState string) []string {
var instID []string
//Values: []*string{aws.String("running"), aws.String("pending")}, - ALSO POSSIBLE TO USE 2 STATES...

	svc := ec2.New(sess)
	fmt.Printf("listing instances with tag %v in: %v\n", HostNameTagFilter, *sess.Config.Region)
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(strings.Join([]string{"*", HostNameTagFilter, "*"}, "")),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String(strings.Join([]string{"*", InstState, "*"}, "")),
				},
		    },
		},
	}
	resp, err := svc.DescribeInstances(params)
	if err != nil {
		fmt.Println("there was an error listing instances in", *sess.Config.Region, err.Error())
		log.Fatal(err.Error())
	}
	//fmt.Printf("%+v\n", *resp) <-TO pring all information


    // resp has all of the response data, pull out instance IDs:
    fmt.Println("> Number of reservation sets: ", len(resp.Reservations))

   for idx, res := range resp.Reservations {
        fmt.Println("  > Number of instances: ", len(res.Instances))
        for _, inst := range resp.Reservations[idx].Instances {
           fmt.Println("    - Instance ID: ", *inst.InstanceId)
           if len(res.Instances) != 0  {
              instID = append(instID, *inst.InstanceId)
            }
       }
    }

    return instID;

}

func startinstance(sess *session.Session, InstanceID string) {


svc := ec2.New(sess)

params := &ec2.StartInstancesInput{
    InstanceIds: []*string {
        aws.String(InstanceID),

    },
    AdditionalInfo: aws.String("String"),
    DryRun:         aws.Bool(true),
}
resp, err := svc.StartInstances(params)

if err != nil {
    // Print the error, cast err to awserr.Error to get the Code and
    // Message from an error.
    fmt.Println(err.Error())
    return
}

// Pretty-print the response data.
fmt.Println(resp)


}



type  appHandler func(http.ResponseWriter, *http.Request) *appError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
        if e := fn(w, r); e != nil { // e is *appError, not os.Error.
                http.Error(w, e.Message, e.Code)
        }
}



type  authHandler func(http.ResponseWriter, *http.Request) *appError

func (fn authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

up := []string{"up"}
down := []string{"down"}

    //TODO: profile and region should be externalized to toml


   // Specify profile for config and region for requests
   sess, err := session.NewSessionWithOptions(session.Options{
        Config: aws.Config{Region: aws.String("us-east-1")},
        Profile: "default",
   })

   if err != nil {
            fmt.Println("failed to create session,", err)
            return
   }

   scale    := r.URL.Query()["scale"];
   scalenum := r.URL.Query().Get("scalenum")
   HostNameTagFilter := r.URL.Query().Get("tag")
   scalenumint, err := strconv.Atoi(scalenum)

   if err != nil {
         fmt.Println(err.Error())
    }


   if scale != nil && scalenum !="" && HostNameTagFilter !="" {


     fmt.Println("QUERY PARAMS %v", scale)


     if scale[0] == up[0] {
            fmt.Println("SCALING UP")
            fmt.Println("PRINTING HOSTS WITH STATE STOPPED")
            InstIDsStopped:=listbyTag(sess,HostNameTagFilter,INSTANCE_STATE_STOPPED)
            fmt.Println("ALL slice of Instances %v", InstIDsStopped)
            http.Error(w, "Scale is UP by => NUM " + scalenum, 200)

           //Scailing by Starting Instances
            ScaleInstances(sess, InstIDsStopped, scalenumint)


     }
    if scale[0] == down[0] {
            fmt.Println("SCALING DOWN")
            fmt.Println("PRINTING HOSTS WITH STATE UP")
            instIDsRunning:=listbyTag(sess,HostNameTagFilter,INSTANCE_STATE_RUNNING)
            fmt.Println("ALL slice of Instances %v", instIDsRunning)
            http.Error(w, "Scale is DOWN by => NUM " + scalenum, 200)
           //Scailing by Starting Instances
            ScaleInstances(sess, instIDsRunning, scalenumint)
    }

    } else {
        http.Error(w, "Dont know scale up or down, or how much to scale, or witch what tag to use in aws to find host", 501)
     }


}


func viewRecord(w http.ResponseWriter, r *http.Request) *appError {
        uid := 22
        fmt.Fprintf(w, "User logged in with uid: %d", uid)
        return nil
}


func init() {
        http.Handle("/view",     appHandler(viewRecord))      // viewRecord is an appHandler function
        http.Handle("/viewAuth", authHandler(viewRecord)) // viewRecord is an authHandler function
}

func main() {
        http.ListenAndServe(":8080", nil)
}
