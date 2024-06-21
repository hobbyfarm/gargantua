package userservice

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	hfv2 "github.com/hobbyfarm/gargantua/v3/pkg/apis/hobbyfarm.io/v2"
	hfClientset "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned"
	hfClientsetv2 "github.com/hobbyfarm/gargantua/v3/pkg/client/clientset/versioned/typed/hobbyfarm.io/v2"
	hfInformers "github.com/hobbyfarm/gargantua/v3/pkg/client/informers/externalversions"
	listerv2 "github.com/hobbyfarm/gargantua/v3/pkg/client/listers/hobbyfarm.io/v2"
	hferrors "github.com/hobbyfarm/gargantua/v3/pkg/errors"
	hflabels "github.com/hobbyfarm/gargantua/v3/pkg/labels"
	"github.com/hobbyfarm/gargantua/v3/pkg/util"
	generalpb "github.com/hobbyfarm/gargantua/v3/protos/general"
	sessionpb "github.com/hobbyfarm/gargantua/v3/protos/session"
	userpb "github.com/hobbyfarm/gargantua/v3/protos/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

const (
	emailIndex = "authc.hobbyfarm.io/user-email-index"
)

type GrpcUserServer struct {
	userpb.UnimplementedUserSvcServer
	userClient    hfClientsetv2.UserInterface
	userIndexer   cache.Indexer
	userLister    listerv2.UserLister
	userSynced    cache.InformerSynced
	sessionClient sessionpb.SessionSvcClient
}

func NewGrpcUserServer(hfClientSet hfClientset.Interface, hfInformerFactory hfInformers.SharedInformerFactory, sessionClient sessionpb.SessionSvcClient) (*GrpcUserServer, error) {
	inf := hfInformerFactory.Hobbyfarm().V2().Users().Informer()
	indexers := map[string]cache.IndexFunc{emailIndex: emailIndexer}
	err := inf.AddIndexers(indexers)
	if err != nil {
		glog.Fatalf("Error adding indexer %v", err)
		return nil, err
	}
	return &GrpcUserServer{
		userClient:    hfClientSet.HobbyfarmV2().Users(util.GetReleaseNamespace()),
		userIndexer:   inf.GetIndexer(),
		userLister:    hfInformerFactory.Hobbyfarm().V2().Users().Lister(),
		userSynced:    inf.HasSynced,
		sessionClient: sessionClient,
	}, nil
}

func emailIndexer(obj interface{}) ([]string, error) {
	user, ok := obj.(*hfv2.User)
	if !ok {
		return []string{}, nil
	}
	return []string{user.Spec.Email}, nil
}

func (u *GrpcUserServer) CreateUser(ctx context.Context, cur *userpb.CreateUserRequest) (*generalpb.ResourceId, error) {
	if len(cur.GetEmail()) == 0 || len(cur.GetPassword()) == 0 {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"error creating user, email or password field blank",
			cur,
		)
	}

	_, err := u.GetUserByEmail(context.Background(), &userpb.GetUserByEmailRequest{Email: cur.GetEmail()})

	if err == nil {
		// the user was found, we should return info
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.AlreadyExists,
			"user %s already exists",
			cur,
			cur.GetEmail(),
		)
	}

	newUser := hfv2.User{}

	hasher := sha256.New()
	hasher.Write([]byte(cur.GetEmail()))
	sha := base32.StdEncoding.WithPadding(-1).EncodeToString(hasher.Sum(nil))[:10]
	id := "u-" + strings.ToLower(sha)
	newUser.Name = id
	newUser.Spec.Email = cur.GetEmail()

	settings := make(map[string]string)
	settings["terminal_theme"] = "default"

	newUser.Spec.Settings = settings

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cur.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			"error while hashing password for email %s",
			cur,
			cur.GetEmail(),
		)
	}

	newUser.Spec.Password = string(passwordHash)
	newUser.Spec.LastLoginTimestamp = time.Now().Format(time.UnixDate)

	_, err = u.userClient.Create(ctx, &newUser, metav1.CreateOptions{})

	if err != nil {
		return &generalpb.ResourceId{}, hferrors.GrpcError(
			codes.Internal,
			"error creating user",
			cur,
		)
	}

	return &generalpb.ResourceId{Id: id}, nil
}

func (u *GrpcUserServer) GetUserById(ctx context.Context, req *generalpb.GetRequest) (*userpb.User, error) {
	user, err := util.GenericHfGetter(ctx, req, u.userClient, u.userLister.Users(util.GetReleaseNamespace()), "user", u.userSynced())
	if err != nil {
		return &userpb.User{}, err
	}

	glog.V(2).Infof("retrieved user %s", user.Name)

	return &userpb.User{
		Id:                  user.Name,
		Uid:                 string(user.UID),
		Email:               user.Spec.Email,
		Password:            user.Spec.Password,
		AccessCodes:         user.Spec.AccessCodes,
		Settings:            user.Spec.Settings,
		LastLoginTimestamp:  user.Spec.LastLoginTimestamp,
		RegisteredTimestamp: user.GetCreationTimestamp().Time.Format(time.UnixDate),
	}, nil
}

func (u *GrpcUserServer) ListUser(ctx context.Context, listOptions *generalpb.ListOptions) (*userpb.ListUsersResponse, error) {
	doLoadFromCache := listOptions.GetLoadFromCache()
	var users []hfv2.User
	var err error
	if !doLoadFromCache {
		var userList *hfv2.UserList
		userList, err = util.ListByHfClient(ctx, listOptions, u.userClient, "users")
		if err == nil {
			users = userList.Items
		}
	} else {
		users, err = util.ListByCache(listOptions, u.userLister, "users", u.userSynced())
	}
	if err != nil {
		glog.Error(err)
		return &userpb.ListUsersResponse{}, err
	}

	preparedUsers := []*userpb.User{} // must be declared this way so as to JSON marshal into [] instead of null
	for _, user := range users {
		preparedUsers = append(preparedUsers, &userpb.User{
			Id:                  user.Name,
			Uid:                 string(user.UID),
			Email:               user.Spec.Email,
			Password:            user.Spec.Password,
			AccessCodes:         user.Spec.AccessCodes,
			Settings:            user.Spec.Settings,
			LastLoginTimestamp:  user.Spec.LastLoginTimestamp,
			RegisteredTimestamp: user.GetCreationTimestamp().Time.Format(time.UnixDate),
		})
	}

	glog.V(2).Infof("listed users")

	return &userpb.ListUsersResponse{Users: preparedUsers}, nil
}

func (u *GrpcUserServer) UpdateUser(ctx context.Context, userRequest *userpb.User) (*userpb.User, error) {
	id := userRequest.GetId()
	if id == "" {
		return &userpb.User{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			userRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.userLister.Users(util.GetReleaseNamespace()).Get(id)
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving user %s",
				userRequest,
				userRequest.GetId(),
			)
		}

		if userRequest.GetEmail() != "" {
			user.Spec.Email = userRequest.GetEmail()
		}

		if userRequest.GetPassword() != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(userRequest.GetPassword()), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("bad")
			}
			user.Spec.Password = string(passwordHash)
		}

		if userRequest.GetAccessCodes() != nil {
			user.Spec.AccessCodes = userRequest.GetAccessCodes()
		}
		if userRequest.GetSettings() != nil {
			user.Spec.Settings = userRequest.GetSettings()
		}

		_, updateErr := u.userClient.Update(ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &userpb.User{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			userRequest,
		)
	}

	return userRequest, nil
}

func (u *GrpcUserServer) UpdateAccessCodes(ctx context.Context, updateAccessCodesRequest *userpb.UpdateAccessCodesRequest) (*userpb.User, error) {
	id := updateAccessCodesRequest.GetId()
	if id == "" {
		return &userpb.User{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"no ID passed in",
			updateAccessCodesRequest,
		)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.userLister.Users(util.GetReleaseNamespace()).Get(id)
		if err != nil {
			glog.Error(err)
			return hferrors.GrpcError(
				codes.Internal,
				"error while retrieving user %s",
				updateAccessCodesRequest,
				updateAccessCodesRequest.GetId(),
			)
		}

		if updateAccessCodesRequest.GetAccessCodes() != nil {
			user.Spec.AccessCodes = updateAccessCodesRequest.GetAccessCodes()
		} else {
			user.Spec.AccessCodes = make([]string, 0)
		}

		_, updateErr := u.userClient.Update(ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		return &userpb.User{}, hferrors.GrpcError(
			codes.Internal,
			"error attempting to update",
			updateAccessCodesRequest,
		)
	}

	return &userpb.User{}, nil
}

func (u *GrpcUserServer) SetLastLoginTimestamp(ctx context.Context, userId *generalpb.ResourceId) (*emptypb.Empty, error) {
	id := userId.GetId()

	if len(id) == 0 {
		return &emptypb.Empty{}, hferrors.GrpcIdNotSpecifiedError(userId)
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		user, err := u.userLister.Users(util.GetReleaseNamespace()).Get(id)
		if err != nil {
			newErr := status.Newf(
				codes.Internal,
				"error while retrieving user %s",
				id,
			)
			newErr, wde := newErr.WithDetails(userId)
			if wde != nil {
				return wde
			}
			glog.Error(err)
			return newErr.Err()
		}

		user.Spec.LastLoginTimestamp = time.Now().Format(time.UnixDate)

		_, updateErr := u.userClient.Update(ctx, user, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		newErr := status.Newf(
			codes.Internal,
			"error attempting to update",
		)
		newErr, wde := newErr.WithDetails(userId)
		if wde != nil {
			return &emptypb.Empty{}, wde
		}
		return &emptypb.Empty{}, newErr.Err()
	}

	return &emptypb.Empty{}, nil
}

func (u *GrpcUserServer) GetUserByEmail(ctx context.Context, gur *userpb.GetUserByEmailRequest) (*userpb.User, error) {
	if len(gur.GetEmail()) == 0 {
		return &userpb.User{}, hferrors.GrpcError(
			codes.InvalidArgument,
			"email passed in was empty",
			gur,
		)
	}

	obj, err := u.userIndexer.ByIndex(emailIndex, gur.GetEmail())
	if err != nil {
		return &userpb.User{}, hferrors.GrpcError(
			codes.Internal,
			"error while retrieving user by e-mail: %s with error: %v",
			gur,
			gur.GetEmail(),
			err,
		)
	}

	if len(obj) < 1 {
		return &userpb.User{}, hferrors.GrpcError(
			codes.NotFound,
			"user not found by email: %s",
			gur,
			gur.GetEmail(),
		)
	}

	user, ok := obj[0].(*hfv2.User)

	if !ok {
		return &userpb.User{}, hferrors.GrpcError(
			codes.Internal,
			"error while converting user found by email to object: %s",
			gur,
			gur.GetEmail(),
		)
	}

	return &userpb.User{
		Id:          user.Name,
		Email:       user.Spec.Email,
		Password:    user.Spec.Password,
		AccessCodes: user.Spec.AccessCodes,
		Settings:    user.Spec.Settings,
	}, nil
}

func (u *GrpcUserServer) DeleteUser(ctx context.Context, userId *generalpb.ResourceId) (*emptypb.Empty, error) {
	id := userId.GetId()
	user, err := util.GenericHfGetter(
		ctx, &generalpb.GetRequest{Id: id},
		u.userClient,
		u.userLister.Users(util.GetReleaseNamespace()),
		"user",
		u.userSynced(),
	)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	sessionList, err := u.sessionClient.ListSession(ctx, &generalpb.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", hflabels.UserLabel, id),
	})

	if err != nil {
		glog.Errorf("error retrieving session list for user %s during delete: %s", id, err)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error retrieving session list for user %s",
			userId,
			userId.GetId(),
		)
	}

	if len(sessionList.Sessions) > 0 {
		// there are sessions present but they may be expired. let's check
		for _, s := range sessionList.Sessions {
			if !s.Status.Finished {
				return &emptypb.Empty{}, hferrors.GrpcError(
					codes.Internal,
					"cannot delete user, existing sessions found",
					userId,
				)
			}
		}

		// getting here means there are sessions present but they are not active
		// let's delete them for cleanliness' sake
		if ok, err := u.deleteSessions(ctx, sessionList.Sessions); !ok {
			glog.Errorf("error deleting old sessions for user %s: %s", id, err)
			return &emptypb.Empty{}, hferrors.GrpcError(
				codes.Internal,
				"cannot delete user, error removing old sessions",
				userId,
			)
		}
	}

	// at this point we have either delete all old sessions, or there were no sessions  to begin with
	// so we should be safe to delete the user

	deleteErr := u.userClient.Delete(ctx, user.Name, metav1.DeleteOptions{})
	if deleteErr != nil {
		glog.Errorf("error deleting user %s: %s", id, deleteErr)
		return &emptypb.Empty{}, hferrors.GrpcError(
			codes.Internal,
			"error deleting user %s",
			userId,
			userId.GetId(),
		)
	}
	return &emptypb.Empty{}, nil
}

func (u *GrpcUserServer) deleteSessions(ctx context.Context, sessions []*sessionpb.Session) (bool, error) {
	for _, s := range sessions {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// @TODO: Use gRPC SessionClient here!
			_, err := u.sessionClient.DeleteSession(ctx, &generalpb.ResourceId{Id: s.Id})
			return err
		})

		if retryErr != nil {
			return false, retryErr
		}
	}

	return true, nil
}
