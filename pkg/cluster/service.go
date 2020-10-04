package cluster

import (
	"fmt"
	"sync"
	"time"

	"github.com/codeready-toolchain/devcluster/pkg/icmock"

	"github.com/codeready-toolchain/devcluster/pkg/auth"

	devclustererr "github.com/codeready-toolchain/devcluster/pkg/errors"
	"github.com/codeready-toolchain/devcluster/pkg/ibmcloud"
	"github.com/codeready-toolchain/devcluster/pkg/log"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	StatusDeleted        = "deleted"
	StatusDeleting       = "deleting"
	StatusFailed         = "failed"
	StatusNormal         = "normal"
	StatusReady          = "ready"
	StatusProvisioning   = "provisioning"
	StatusFailedToDelete = "failed to delete"
	StatusExpired        = "expired"
	StatusFailedToExpire = "failed to expire"
)

// Request represents a cluster request
type Request struct {
	ID            string
	Requested     int // Number of clusters requested
	Created       int64
	Status        string
	Error         string
	RequestedBy   string
	Zone          string
	DeleteInHours int
	NoSubnet      bool
}

// Request represents a cluster request with detailed information about all request clusters
type RequestWithClusters struct {
	Request  `json:",inline"`
	Clusters []Cluster
}

type Cluster struct {
	ID        string
	RequestID string
	Name      string
	URL       string
	MasterURL string
	Status    string
	Error     string
	User      User
}

type User struct {
	ID            string // <iam_object>.user_id & <cloud_direct_object>.username
	CloudDirectID string // <cloud_direct_object>.id
	Email         string
	Password      string
	ClusterID     string
	PolicyID      string
	Recycled      int64 // last recycle timestamp
}

var DefaultClusterService *ClusterService

// ClusterService represents a registry of all cluster resources
type ClusterService struct {
	IbmCloudClient ibmcloud.ICClient
	Config         ibmcloud.Configuration
}

func InitDefaultClusterService(config ibmcloud.Configuration) {
	DefaultClusterService = &ClusterService{
		IbmCloudClient: icmock.NewMockIBMCloudClient(),
		Config:         config,
	}
}

func (s *ClusterService) Requests() ([]Request, error) {
	return getAllRequests()
}

func (s *ClusterService) GetZones() ([]ibmcloud.Location, error) {
	return s.IbmCloudClient.GetZones()
}

func (s *ClusterService) GetRequestWithClusters(requestID string) (*RequestWithClusters, error) {
	request, err := getRequest(requestID)
	if err != nil {
		return nil, err
	}
	if request == nil {
		// Not found
		return nil, nil
	}
	clusters, err := getClusters(requestID)
	for i, _ := range clusters {
		user, err := GetUserByClusterID(clusters[i].ID)
		if err != nil {
			if devclustererr.IsNotFound(err) {
				continue // Ignore not found users
			}
			return nil, err
		}
		clusters[i].User = *user
	}
	if err != nil {
		return nil, err
	}
	return &RequestWithClusters{
		Request:  *request,
		Clusters: clusters,
	}, nil
}

// CreateNewRequest creates a new request and starts provisioning clusters
func (s *ClusterService) CreateNewRequest(requestedBy string, n int, zone string, deleteInHours int, noSubnet bool) (Request, error) {
	r := Request{
		ID:            uuid.NewV4().String(),
		Requested:     n,
		Created:       time.Now().Unix(),
		Status:        StatusProvisioning,
		RequestedBy:   requestedBy,
		Zone:          zone,
		DeleteInHours: deleteInHours,
		NoSubnet:      noSubnet,
	}

	err := insertRequest(r)
	if err != nil {
		return Request{}, errors.Wrap(err, "unable to start new request")
	}
	for i := 0; i < r.Requested; i++ {
		time.Sleep(time.Millisecond)
		go func() {
			err := s.provisionNewCluster(r)
			if err != nil {
				log.Error(nil, err, "unable to provision a cluster")
			}
		}()
	}

	return r, nil
}

// DeleteCluster deletes the cluster with the given ID
func (s *ClusterService) DeleteCluster(id string) error {
	if err := s.IbmCloudClient.DeleteCluster(id); err != nil {
		return err
	}
	c, err := getCluster(id)
	if err != nil {
		return err
	}
	c.Error = ""
	c.Status = StatusDeleted
	if err := s.recycleUser(id); err != nil {
		return err
	}
	return replaceCluster(*c)
}

// ResumeProvisioningRequests load requests that are still provisioning and wait for their clusters to be ready to update the status
func (s *ClusterService) ResumeProvisioningRequests() error {
	requests, err := getRequestsWithStatus(StatusProvisioning)
	if err != nil {
		return err
	}
	for _, r := range requests {
		resumeRequest := r // need to use a copy in goroutine
		clusters, err := getClusters(resumeRequest.ID)
		if err != nil {
			return err
		}
		// Update provisioning clusters
		for _, cluster := range clusters {
			resumeCluster := cluster // need to use a copy in goroutine
			if clusterProvisioningPending(resumeCluster) {
				go func() {
					log.Infof(nil, "resuming provisioning cluster %s", resumeCluster.Name)
					err := s.waitForClusterToBeReady(resumeRequest, resumeCluster.ID, resumeCluster.Name)
					if err != nil {
						log.Error(nil, err, "unable to provision a cluster")
					}
				}()
			}
		}
		// Start provisioning missing clusters
		count := len(clusters)
		if count < resumeRequest.Requested {
			for i := count; i < resumeRequest.Requested; i++ {
				go func() {
					err := s.provisionNewCluster(resumeRequest)
					if err != nil {
						log.Error(nil, err, "unable to provision a cluster")
					}
				}()
			}
		}
	}

	return nil
}

// StartDeletingExpiredClusters starts a goroutine to check expired clusters every n seconds and delete them
func (s *ClusterService) StartDeletingExpiredClusters(intervalInSec int) {
	go func() {
		for {
			reqs, err := getAllRequests()
			if err != nil {
				log.Error(nil, err, "unable to get request to check expired clusters")
			} else {
				for _, r := range reqs {
					if r.Status != "expired" && expired(r) { // cluster is expired but the status is not yet set to "expired"
						clusters, err := getClusters(r.ID)
						if err != nil {
							log.Error(nil, err, "unable to get clusters to check expired")
						} else {
							allDeleted := true
							for _, c := range clusters {
								if c.Status != StatusDeleted && c.Status != StatusDeleting {
									// Delete the expired cluster
									err := s.DeleteCluster(c.ID)
									if err != nil {
										// Set the error status for the cluster
										clusterFailedToDelete(c, err)
										allDeleted = false
									}
								}
							}
							if allDeleted {
								// All clusters deleted. Mark the request as expired.
								err = updateRequestStatus(r.ID, StatusExpired, "")
							} else {
								// Failed to delete at least one cluster. Mark the request as failed to expire.
								err = updateRequestStatus(r.ID, StatusFailedToExpire, "unable to delete some clusters")
							}
							if err != nil {
								log.Error(nil, err, "unable to update request status")
							}
						}
					}
				}
			}
			time.Sleep(time.Duration(intervalInSec) * time.Second)
		}
	}()
}

func expired(r Request) bool {
	return time.Unix(r.Created, 0).Add(time.Duration(r.DeleteInHours) * time.Hour).Before(time.Now())
}

// provisionNewCluster creates one new cluster
func (s *ClusterService) provisionNewCluster(r Request) error {
	name := auth.GenerateShortID("redhat")
	var id string
	var err error
	// Try to create a cluster. If failing then we will make six attempts for one minute before giving up.
	for i := 0; i < 6; i++ {
		id, err = s.IbmCloudClient.CreateCluster(name, r.Zone, r.NoSubnet)
		if err == nil {
			if err := s.assignUser(id); err != nil {
				log.Error(nil, err, "unable to assign a user to the cluster")
				return err
			}
			break
		}
		log.Error(nil, err, "unable to create cluster")
		time.Sleep(10 * time.Second)
	}
	if err != nil {
		// Set request status to failed and break
		r.Status = StatusFailed
		r.Error = err.Error()
		err := updateRequestStatus(r.ID, StatusFailed, err.Error())
		return err
	}
	log.Infof(nil, "starting provisioning cluster %s", name)
	return s.waitForClusterToBeReady(r, id, name)
}

// Controls access to the user pool for assigning clusters
var clusterAssigneeMux sync.Mutex

// assignUser picks a free user from the user pool and grands access to the cluster to that user
// by creating an Access Policy.
func (s *ClusterService) assignUser(clusterID string) error {
	user, err := s.obtainFreeUser(clusterID)
	if err != nil {
		return err
	}
	policyID, err := s.IbmCloudClient.CreateAccessPolicy(s.Config.GetIBMCloudAccountID(), user.ID, clusterID)
	if err != nil {
		rollBackClusterAssigment(*user)
		log.Error(nil, err, fmt.Sprintf("unable to create access policy for user ID: %s", user.ID))
		return err
	}
	user.PolicyID = policyID
	return replaceUser(*user)
}

// rollBackClusterAssigment rolls back cluster assigment for the user
func rollBackClusterAssigment(user User) {
	user.ClusterID = ""
	if e := replaceUser(user); e != nil {
		log.Error(nil, e, fmt.Sprintf("unable to roll back cluster assigment for the user with id: %s", user.ID))
	}
}

// obtainFreeUser obtains a free user from the user pool and sets the cluster ID to that user so it can not be assigned to another cluster
func (s *ClusterService) obtainFreeUser(clusterID string) (*User, error) {
	clusterAssigneeMux.Lock()
	defer clusterAssigneeMux.Unlock()
	user, err := getUserWithoutCluster()
	if err != nil {
		return nil, err
	}
	user.ClusterID = clusterID

	return user, replaceUser(*user)
}

// recycleUser change the password of the user assigned to the cluster and returns that user to the user pool
// so it can be assigned to another cluster.
func (s *ClusterService) recycleUser(clusterID string) error {
	user, err := GetUserByClusterID(clusterID)
	if err != nil {
		return err
	}
	if err := s.IbmCloudClient.DeleteAccessPolicy(user.PolicyID); err != nil {
		return err
	}
	cloudDirUser, err := s.IbmCloudClient.UpdateCloudDirectoryUserPassword(user.CloudDirectID)
	if err != nil {
		log.Error(nil, err, fmt.Sprintf("unable to update cloud directory user password for user: %s", user.ID))
		return err
	}
	user.PolicyID = ""
	user.ClusterID = ""
	user.Password = cloudDirUser.Password
	user.Recycled = time.Now().Unix()

	return replaceUser(*user)
}

// waitForClusterToBeReady for the cluster to be ready
func (s *ClusterService) waitForClusterToBeReady(r Request, clusterID, clusterName string) error {
	in3Hours := time.Now().Add(3 * time.Hour)
	for time.Now().Before(in3Hours) { // timeout in three hours
		c, err := s.IbmCloudClient.GetCluster(clusterID)
		if err != nil {
			log.Error(nil, err, "unable to get cluster")
			if devclustererr.IsNotFound(err) {
				// set the state to "deleted" but only if it's not in the "deleted" state already (in case of manual deletion) and return.
				// otherwise set the status to "deleted" with the error message from IBM Cloud and try again in s.config.GetIBMCloudApiCallRetrySec() seconds.
				cl, e := getCluster(clusterID)
				if e != nil {
					return e
				}
				if cl == nil || cl.Status != StatusDeleted {
					e := clusterFailed(err, StatusDeleted, clusterID, clusterName, r.ID)
					if e != nil {
						return e
					}
				} else {
					return nil
				}
			} else {
				err := clusterFailed(err, "failed", clusterID, clusterName, r.ID)
				if err != nil {
					return err
				}
			}
			// Do not return. Try again in s.config.GetIBMCloudApiCallRetrySec() seconds.
		} else {
			clusterToAdd := convertCluster(*c, r.ID)
			err := replaceCluster(clusterToAdd)
			if err != nil {
				return err
			}
			if clusterReady(clusterToAdd) { // Ready
				err := setRequestStatusToSuccessIfDone(r)
				if err != nil {
					return err
				}
				// TODO add user
				break
			}
		}
		time.Sleep(time.Duration(s.Config.GetIBMCloudApiCallRetrySec()) * time.Second)
	}
	return nil
}

// CreateUsers creates n number of users
// For example if n == 3 and startIndex == 1000 then the following users will be created:
// rd-dev-1001, rd-dev-1002, rd-dev-1003
func (s *ClusterService) CreateUsers(n, startIndex int) ([]User, error) {
	users := make([]User, 0, 0)
	for i := startIndex + 1; i <= startIndex+n; i++ {
		cdu, err := s.IbmCloudClient.CreateCloudDirectoryUser(fmt.Sprintf("rh-dev-%d", i))
		if err != nil {
			return nil, err
		}
		user := User{
			ID:            cdu.Username,
			CloudDirectID: cdu.ID,
			Email:         cdu.Email(),
			Password:      cdu.Password,
		}
		err = insertUser(user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *ClusterService) Users() ([]User, error) {
	return getAllUsers()
}

func clusterReady(c Cluster) bool {
	return c.Status == StatusNormal && c.URL != "" && c.MasterURL != ""
}

// clusterProvisioningPending returns true if cluster is still provisioning
func clusterProvisioningPending(c Cluster) bool {
	return !clusterReady(c) && c.Status != StatusFailed && c.Status != StatusDeleted
}

func clusterFailed(clErr error, status, id, name, reqID string) error {
	clToUpdate, err := getCluster(id)
	if err != nil {
		return err
	}
	if clToUpdate == nil {
		clToUpdate = &Cluster{
			ID:        id,
			Name:      name,
			RequestID: reqID,
		}
	}
	clToUpdate.Error = clErr.Error()
	clToUpdate.Status = status
	return replaceCluster(*clToUpdate)
}

func clusterFailedToDelete(c Cluster, e error) {
	log.Error(nil, e, "unable to delete expired cluster")
	err := replaceCluster(Cluster{
		ID:        c.ID,
		RequestID: c.RequestID,
		Name:      c.Name,
		URL:       c.URL,
		MasterURL: c.MasterURL,
		Status:    StatusFailedToDelete,
		Error:     e.Error(),
	})
	if err != nil {
		log.Error(nil, err, "unable to update status for failed to delete cluster")
	}
}

func convertCluster(from ibmcloud.Cluster, requestID string) Cluster {
	console := from.Ingress.Hostname
	if console != "" {
		console = "https://console-openshift-console." + console
	}
	return Cluster{
		ID:        from.ID,
		URL:       console,
		MasterURL: from.MasterURL,
		Status:    from.State,
		Name:      from.Name,
		RequestID: requestID,
	}
}
