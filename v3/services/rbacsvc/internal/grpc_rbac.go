package rbac

import (
	"context"

	"github.com/golang/glog"
	"github.com/hobbyfarm/gargantua/v3/pkg/errors"
	authrProto "github.com/hobbyfarm/gargantua/v3/protos/authr"
	"github.com/hobbyfarm/gargantua/v3/protos/general"
	rbacProto "github.com/hobbyfarm/gargantua/v3/protos/rbac"
	"google.golang.org/grpc/codes"
)

func (rs *GrpcRbacServer) Grants(c context.Context, gr *rbacProto.GrantRequest) (*authrProto.AuthRResponse, error) {
	as, err := rs.userIndex.GetAccessSet(gr.GetUserName())
	if err != nil {
		glog.Errorf("failed to retrieve access set for user %s", gr.GetUserName())
		return &authrProto.AuthRResponse{}, errors.GrpcError(
			codes.Internal,
			"failed to retrieve access set for user %s",
			gr,
			gr.GetUserName(),
		)
	}

	return &authrProto.AuthRResponse{Success: Grants(gr.GetPermission(), as)}, nil
}

func (rs *GrpcRbacServer) GetAccessSet(c context.Context, uid *general.ResourceId) (*rbacProto.AccessSet, error) {
	return rs.userIndex.GetAccessSet(uid.GetId())
}

func (rs *GrpcRbacServer) GetHobbyfarmRoleBindings(c context.Context, uid *general.ResourceId) (*rbacProto.RoleBindings, error) {
	rbs, err := rs.userIndex.getPreparedRoleBindings(uid.GetId())
	if err != nil {
		glog.Errorf("failed to retrieve rolebindings for user %s with error: %s", uid.GetId(), err.Error())
		return &rbacProto.RoleBindings{}, errors.GrpcError(
			codes.Internal,
			"failed to retrieve rolebindings for user %s",
			uid,
			uid.GetId(),
		)
	}
	return rbs, nil
}
