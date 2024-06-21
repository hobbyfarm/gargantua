package rbac

import (
	"context"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/errors"
	authrpb "github.com/hobbyfarm/gargantua/v3/protos/authr"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacpb "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
)

func (rs *GrpcRbacServer) Grants(c context.Context, gr *rbacpb.GrantRequest) (*authrpb.AuthRResponse, error) {
	as, err := rs.userIndex.GetAccessSet(gr.GetUserName())
	if err != nil {
		glog.Errorf("failed to retrieve access set for user %s", gr.GetUserName())
		return &authrpb.AuthRResponse{}, errors.GrpcError(
			codes.Internal,
			"failed to retrieve access set for user %s",
			gr,
			gr.GetUserName(),
		)
	}

	return &authrpb.AuthRResponse{Success: Grants(gr.GetPermission(), as)}, nil
}

func (rs *GrpcRbacServer) GetAccessSet(c context.Context, uid *generalpb.ResourceId) (*rbacpb.AccessSet, error) {
	return rs.userIndex.GetAccessSet(uid.GetId())
}

func (rs *GrpcRbacServer) GetHobbyfarmRoleBindings(c context.Context, uid *generalpb.ResourceId) (*rbacpb.RoleBindings, error) {
	rbs, err := rs.userIndex.getPreparedRoleBindings(uid.GetId())
	if err != nil {
		glog.Errorf("failed to retrieve rolebindings for user %s with error: %s", uid.GetId(), err.Error())
		return &rbacpb.RoleBindings{}, errors.GrpcError(
			codes.Internal,
			"failed to retrieve rolebindings for user %s",
			uid,
			uid.GetId(),
		)
	}
	return rbs, nil
}
