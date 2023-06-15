package authrservice

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/pkg/microservices"
	"github.com/hobbyfarm/gargantua/pkg/rbac"
	"github.com/hobbyfarm/gargantua/pkg/util"
	authrProto "github.com/hobbyfarm/gargantua/protos/authr"
	rbacProto "github.com/hobbyfarm/gargantua/protos/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	OperatorAnd = rbac.OperatorAND
	OperatorOr  = rbac.OperatorOR
)

type GrpcAuthRServer struct {
	authrProto.UnimplementedAuthRServer
	tlsCaPath string
}

func NewGrpcAuthRServer(tlsCaPath string) *GrpcAuthRServer {
	return &GrpcAuthRServer{tlsCaPath: tlsCaPath}
}

// This function authorizes the user by using impersonation as an additional security layer.
// After impersonation, the user must also authorize himself against the rbac-service.
// If the authorization fails, this method should always return an AuthRResponse with Success = false AND an error
func (a *GrpcAuthRServer) AuthR(c context.Context, ar *authrProto.AuthRRequest) (*authrProto.AuthRResponse, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("error with in cluster config: %s", err)
	}

	// Create a Kubernetes API clientset.
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("error while creating kubernetes client: %s", err)
	}

	// Establish connection to rbac service
	rbacConn, err := microservices.EstablishConnection("rbac-service", a.tlsCaPath)
	if err != nil {
		glog.Error("failed connecting to service rbac-service")
		msg := "rbac-service unreachable: "
		return a.returnResponseFailedAuthrWithError(ar, msg, err)
	}
	rbacClient := rbacProto.NewRbacSvcClient(rbacConn)
	defer rbacConn.Close()

	// Set impersonated username
	iu := ar.GetUserName()

	if ar.GetRequest().GetOperator() == OperatorAnd || ar.GetRequest().GetOperator() == "" {
		for _, p := range ar.GetRequest().GetPermissions() {
			// Create the SubjectAccessReview request.
			sar := a.createSubjectAccessReview(iu, util.GetReleaseNamespace(), p.GetVerb(), p.GetApiGroup(), p.GetResource())
			// Perform the SubjectAccessReview request.
			result, err := clientset.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
			if err != nil {
				glog.Fatalf("failed to create subject access review: %s", err)
				msg := "error while performing the SubjectAccessReview request:"
				return a.returnResponseFailedAuthrWithError(ar, msg, err)
			}

			rbacAuthGrant, err := rbacClient.Grants(c, &rbacProto.GrantRequest{UserName: iu, Permission: p})
			if err != nil {
				if s, ok := status.FromError(err); ok {
					details := s.Details()[0].(*rbacProto.GrantRequest)
					glog.Errorf("could not perform auth grant for user %s: %s", details.UserName, s.Message())
					glog.Infof("auth grant failed for permission with apiGroup=%s, resource=%s and verb=%s",
						details.GetPermission().GetApiGroup(), details.GetPermission().GetResource(), details.GetPermission().GetVerb())
					msg := "could not perform auth grant: "
					return a.returnResponseFailedAuthrWithError(ar, msg, s.Err())
				}
				msg := "could not perform auth grant: "
				return a.returnResponseFailedAuthrWithError(ar, msg, err)
			}

			if !result.Status.Allowed || !rbacAuthGrant.Success {
				// Return the authorization decision.
				glog.Infof("User %s is not authorized to perform this request", iu)
				return &authrProto.AuthRResponse{
					Success: false,
				}, fmt.Errorf("permission denied")
			}
		}

		// if we get here, AND has succeeded
		return &authrProto.AuthRResponse{
			Success: true,
		}, nil
	} else {
		// operator AND, all need to match
		for _, p := range ar.GetRequest().GetPermissions() {
			// Create the SubjectAccessReview request.
			sar := a.createSubjectAccessReview(iu, util.GetReleaseNamespace(), p.GetVerb(), p.GetApiGroup(), p.GetResource())

			// Perform the SubjectAccessReview request.
			result, err := clientset.AuthorizationV1().SubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
			if err != nil {
				glog.Fatalf("failed to create subject access review: %s", err)
				msg := "error while performing the SubjectAccessReview request:"
				return a.returnResponseFailedAuthrWithError(ar, msg, err)
			}

			rbacAuthGrant, err := rbacClient.Grants(c, &rbacProto.GrantRequest{UserName: iu, Permission: p})
			if err != nil {
				if s, ok := status.FromError(err); ok {
					details := s.Details()[0].(*rbacProto.GrantRequest)
					glog.Errorf("could not perform auth grant for user %s: %s", details.UserName, s.Message())
					glog.Infof("auth grant failed for permission with apiGroup=%s, resource=%s and verb=%s",
						details.GetPermission().GetApiGroup(), details.GetPermission().GetResource(), details.GetPermission().GetVerb())
					msg := "could not perform auth grant: "
					return a.returnResponseFailedAuthrWithError(ar, msg, s.Err())
				}
				msg := "could not perform auth grant: "
				return a.returnResponseFailedAuthrWithError(ar, msg, err)
			}

			if result.Status.Allowed && rbacAuthGrant.Success {
				// Return the authorization decision.
				return &authrProto.AuthRResponse{
					Success: true,
				}, nil
			}
		}
	}

	return &authrProto.AuthRResponse{
		Success: false,
	}, fmt.Errorf("permission denied")
}

func (a *GrpcAuthRServer) createSubjectAccessReview(userName string, releaseNamespace string, verb string, apiGroup string, resource string) *v1.SubjectAccessReview {
	sar := &v1.SubjectAccessReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubjectAccessReview",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		Spec: v1.SubjectAccessReviewSpec{
			ResourceAttributes: &v1.ResourceAttributes{
				Namespace: releaseNamespace,
				Verb:      verb,
				Group:     apiGroup,
				Resource:  resource,
			},
			User: userName,
		},
	}
	return sar
}

func (a *GrpcAuthRServer) returnResponseFailedAuthrWithError(ar *authrProto.AuthRRequest, msg string, err error) (*authrProto.AuthRResponse, error) {
	newErr := status.Newf(
		codes.Internal,
		"%s %s",
		msg,
		err,
	)
	newErr, wde := newErr.WithDetails(ar)
	if wde != nil {
		return &authrProto.AuthRResponse{
			Success: false,
		}, wde
	}
	return &authrProto.AuthRResponse{
		Success: false,
	}, newErr.Err()
}
