package repocreds

import (
	"context"
	"reflect"

	"github.com/argoproj/argo-cd/v3/util/argo"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	repocredspkg "github.com/argoproj/argo-cd/v3/pkg/apiclient/repocreds"
	appsv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/util/db"
	"github.com/argoproj/argo-cd/v3/util/rbac"
)

// Server provides a Repository service
type Server struct {
	db  db.ArgoDB
	enf *rbac.Enforcer
}

// NewServer returns a new instance of the Repository service
func NewServer(
	db db.ArgoDB,
	enf *rbac.Enforcer,
) *Server {
	return &Server{
		db:  db,
		enf: enf,
	}
}

// ListRepositoryCredentials returns a list of all configured repository credential sets
func (s *Server) ListRepositoryCredentials(ctx context.Context, _ *repocredspkg.RepoCredsQuery) (*appsv1.RepoCredsList, error) {
	urls, err := s.db.ListRepositoryCredentials(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]appsv1.RepoCreds, 0)
	for _, url := range urls {
		if s.enf.Enforce(ctx.Value("claims"), rbac.ResourceRepositories, rbac.ActionGet, url) {
			repo, err := s.db.GetRepositoryCredentials(ctx, url)
			if err != nil {
				return nil, err
			}
			if repo != nil {
				items = append(items, appsv1.RepoCreds{
					URL:      url,
					Username: repo.Username,
				})
			}
		}
	}
	return &appsv1.RepoCredsList{Items: items}, nil
}

// ListWriteRepositoryCredentials returns a list of all configured repository credential sets
func (s *Server) ListWriteRepositoryCredentials(ctx context.Context, _ *repocredspkg.RepoCredsQuery) (*appsv1.RepoCredsList, error) {
	urls, err := s.db.ListRepositoryCredentials(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]appsv1.RepoCreds, 0)
	for _, url := range urls {
		if s.enf.Enforce(ctx.Value("claims"), rbac.ResourceWriteRepositories, rbac.ActionGet, url) {
			repo, err := s.db.GetWriteRepositoryCredentials(ctx, url)
			if err != nil {
				return nil, err
			}
			if repo != nil && repo.Password != "" {
				items = append(items, appsv1.RepoCreds{
					URL:      url,
					Username: repo.Username,
				})
			}
		}
	}
	return &appsv1.RepoCredsList{Items: items}, nil
}

// CreateRepositoryCredentials creates a new credential set in the configuration
func (s *Server) CreateRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsCreateRequest) (*appsv1.RepoCreds, error) {
	if q.Creds == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing payload in request")
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceRepositories, rbac.ActionCreate, q.Creds.URL); err != nil {
		return nil, err
	}

	r := q.Creds

	if r.URL == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify URL")
	}

	_, err := s.db.CreateRepositoryCredentials(ctx, r)
	if status.Convert(err).Code() == codes.AlreadyExists {
		// act idempotent if existing spec matches new spec
		existing, getErr := s.db.GetRepositoryCredentials(ctx, r.URL)
		if getErr != nil {
			return nil, status.Errorf(codes.Internal, "unable to check existing repository credentials details: %v", getErr)
		}

		switch {
		case reflect.DeepEqual(existing, r):
			err = nil
		case q.Upsert:
			return s.UpdateRepositoryCredentials(ctx, &repocredspkg.RepoCredsUpdateRequest{Creds: r})
		default:
			return nil, status.Error(codes.InvalidArgument, argo.GenerateSpecIsDifferentErrorMessage("repository credentials", existing, r))
		}
	}
	return &appsv1.RepoCreds{URL: r.URL}, err
}

// CreateWriteRepositoryCredentials creates a new credential set in the configuration
func (s *Server) CreateWriteRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsCreateRequest) (*appsv1.RepoCreds, error) {
	if q.Creds == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing payload in request")
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceWriteRepositories, rbac.ActionCreate, q.Creds.URL); err != nil {
		return nil, err
	}

	r := q.Creds

	if r.URL == "" {
		return nil, status.Errorf(codes.InvalidArgument, "must specify URL")
	}

	_, err := s.db.CreateWriteRepositoryCredentials(ctx, r)
	if status.Convert(err).Code() == codes.AlreadyExists {
		// act idempotent if existing spec matches new spec
		existing, getErr := s.db.GetWriteRepositoryCredentials(ctx, r.URL)
		if getErr != nil {
			return nil, status.Errorf(codes.Internal, "unable to check existing repository credentials details: %v", getErr)
		}

		switch {
		case reflect.DeepEqual(existing, r):
			err = nil
		case q.Upsert:
			return s.UpdateWriteRepositoryCredentials(ctx, &repocredspkg.RepoCredsUpdateRequest{Creds: r})
		default:
			return nil, status.Error(codes.InvalidArgument, argo.GenerateSpecIsDifferentErrorMessage("repository credentials", existing, r))
		}
	}
	return &appsv1.RepoCreds{URL: r.URL}, err
}

// UpdateRepositoryCredentials updates a repository credential set
func (s *Server) UpdateRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsUpdateRequest) (*appsv1.RepoCreds, error) {
	if q.Creds == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing payload in request")
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceRepositories, rbac.ActionUpdate, q.Creds.URL); err != nil {
		return nil, err
	}
	_, err := s.db.UpdateRepositoryCredentials(ctx, q.Creds)
	return &appsv1.RepoCreds{URL: q.Creds.URL}, err
}

// UpdateWriteRepositoryCredentials updates a repository credential set
func (s *Server) UpdateWriteRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsUpdateRequest) (*appsv1.RepoCreds, error) {
	if q.Creds == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing payload in request")
	}
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceWriteRepositories, rbac.ActionUpdate, q.Creds.URL); err != nil {
		return nil, err
	}
	_, err := s.db.UpdateWriteRepositoryCredentials(ctx, q.Creds)
	return &appsv1.RepoCreds{URL: q.Creds.URL}, err
}

// DeleteRepositoryCredentials removes a credential set from the configuration
func (s *Server) DeleteRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsDeleteRequest) (*repocredspkg.RepoCredsResponse, error) {
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceRepositories, rbac.ActionDelete, q.Url); err != nil {
		return nil, err
	}

	err := s.db.DeleteRepositoryCredentials(ctx, q.Url)
	return &repocredspkg.RepoCredsResponse{}, err
}

// DeleteWriteRepositoryCredentials removes a credential set from the configuration
func (s *Server) DeleteWriteRepositoryCredentials(ctx context.Context, q *repocredspkg.RepoCredsDeleteRequest) (*repocredspkg.RepoCredsResponse, error) {
	if err := s.enf.EnforceErr(ctx.Value("claims"), rbac.ResourceWriteRepositories, rbac.ActionDelete, q.Url); err != nil {
		return nil, err
	}

	err := s.db.DeleteWriteRepositoryCredentials(ctx, q.Url)
	return &repocredspkg.RepoCredsResponse{}, err
}
