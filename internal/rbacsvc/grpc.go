package rbac

import (
	"context"

	"github.com/golang/glog"
	authrProto "github.com/hobbyfarm/gargantua/protos/authr"
	rbacProto "github.com/hobbyfarm/gargantua/protos/rbac"
	userProto "github.com/hobbyfarm/gargantua/protos/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/informers"
)

const (
	VerbList   = "list"
	VerbGet    = "get"
	VerbCreate = "create"
	VerbUpdate = "update"
	VerbDelete = "delete"
	VerbWatch  = "watch"
)

type GrpcRbacServer struct {
	rbacProto.UnimplementedRbacSvcServer
	userIndex  *Index
	groupIndex *Index
}

func NewGrpcRbacServer(namespace string, kubeInformerFactory informers.SharedInformerFactory) (*GrpcRbacServer, error) {
	userIndex, err := NewIndex("User", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	groupIndex, err := NewIndex("Group", namespace, kubeInformerFactory)
	if err != nil {
		return nil, err
	}

	return &GrpcRbacServer{
		userIndex:  userIndex,
		groupIndex: groupIndex,
	}, nil
}

func (rs *GrpcRbacServer) Grants(c context.Context, gr *rbacProto.GrantRequest) (*authrProto.AuthRResponse, error) {
	as, err := rs.userIndex.GetAccessSet(gr.GetUserName())
	if err != nil {
		err := status.Newf(
			codes.Internal,
			"failed to retrieve access set for user %s",
			gr.GetUserName(),
		)

		err, wde := err.WithDetails(gr)
		if wde != nil {
			return &authrProto.AuthRResponse{Success: false}, wde
		}
		glog.Errorf("failed to retrieve access set for user %s", gr.GetUserName())
		return &authrProto.AuthRResponse{Success: false}, err.Err()
	}

	return &authrProto.AuthRResponse{Success: Grants(gr.GetPermission(), as)}, nil
}

func (rs *GrpcRbacServer) GetAccessSet(c context.Context, uid *userProto.UserId) (*rbacProto.AccessSet, error) {
	return rs.userIndex.GetAccessSet(uid.GetId())
}

func (rs *GrpcRbacServer) GetHobbyfarmRoleBindings(c context.Context, uid *userProto.UserId) (*rbacProto.RoleBindings, error) {
	rbs, err := rs.userIndex.getPreparedRoleBindings(uid.GetId())
	if err != nil {
		glog.Errorf("failed to retrieve rolebindings for user %s with error: %s", uid.GetId(), err.Error())
		err := status.Newf(
			codes.Internal,
			"failed to retrieve rolebindings for user %s",
			uid.GetId(),
		)

		err, wde := err.WithDetails(uid)
		if wde != nil {
			return nil, wde
		}

		return nil, err.Err()
	}
	return rbs, nil
}
