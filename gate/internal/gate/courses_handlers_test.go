package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/XaviFP/toshokan/course/api/proto/v1"
	pbUser "github.com/XaviFP/toshokan/user/api/proto/v1"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockCourseAPIClient is a mock implementation of pb.CourseAPIClient
type mockCourseAPIClient struct {
	mock.Mock
}

func (m *mockCourseAPIClient) GetCourse(ctx context.Context, req *pb.GetCourseRequest, opts ...grpc.CallOption) (*pb.GetCourseResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetCourseResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetLesson(ctx context.Context, req *pb.GetLessonRequest, opts ...grpc.CallOption) (*pb.GetLessonResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetLessonResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetLessons(ctx context.Context, req *pb.GetLessonsRequest, opts ...grpc.CallOption) (*pb.GetLessonsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetLessonsResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetFocusedLessons(ctx context.Context, req *pb.GetFocusedLessonsRequest, opts ...grpc.CallOption) (*pb.GetFocusedLessonsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetFocusedLessonsResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetEnrolledCourses(ctx context.Context, req *pb.GetEnrolledCoursesRequest, opts ...grpc.CallOption) (*pb.GetEnrolledCoursesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetEnrolledCoursesResponse), args.Error(1)
}

func (m *mockCourseAPIClient) EnrollUser(ctx context.Context, req *pb.EnrollUserRequest, opts ...grpc.CallOption) (*pb.EnrollUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.EnrollUserResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetUserProgress(ctx context.Context, req *pb.GetUserProgressRequest, opts ...grpc.CallOption) (*pb.GetUserProgressResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetUserProgressResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetLessonState(ctx context.Context, req *pb.GetLessonStateRequest, opts ...grpc.CallOption) (*pb.GetLessonStateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetLessonStateResponse), args.Error(1)
}

func (m *mockCourseAPIClient) AnswerCards(ctx context.Context, req *pb.AnswerCardsRequest, opts ...grpc.CallOption) (*pb.AnswerCardsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.AnswerCardsResponse), args.Error(1)
}

func (m *mockCourseAPIClient) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest, opts ...grpc.CallOption) (*pb.CreateCourseResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.CreateCourseResponse), args.Error(1)
}

func (m *mockCourseAPIClient) CreateLesson(ctx context.Context, req *pb.CreateLessonRequest, opts ...grpc.CallOption) (*pb.CreateLessonResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.CreateLessonResponse), args.Error(1)
}

func (m *mockCourseAPIClient) UpdateCourse(ctx context.Context, req *pb.UpdateCourseRequest, opts ...grpc.CallOption) (*pb.UpdateCourseResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.UpdateCourseResponse), args.Error(1)
}

func (m *mockCourseAPIClient) UpdateLesson(ctx context.Context, req *pb.UpdateLessonRequest, opts ...grpc.CallOption) (*pb.UpdateLessonResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.UpdateLessonResponse), args.Error(1)
}

func (m *mockCourseAPIClient) SyncState(ctx context.Context, req *pb.SyncStateRequest, opts ...grpc.CallOption) (*pb.SyncStateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.SyncStateResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetCourseDeckIds(ctx context.Context, req *pb.GetCourseDeckIdsRequest, opts ...grpc.CallOption) (*pb.GetCourseDeckIdsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetCourseDeckIdsResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetLessonDeckIds(ctx context.Context, req *pb.GetLessonDeckIdsRequest, opts ...grpc.CallOption) (*pb.GetLessonDeckIdsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetLessonDeckIdsResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetStudentsProgress(ctx context.Context, req *pb.GetStudentsProgressRequest, opts ...grpc.CallOption) (*pb.GetStudentsProgressResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetStudentsProgressResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetUserState(ctx context.Context, req *pb.GetUserStateRequest, opts ...grpc.CallOption) (*pb.GetUserStateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetUserStateResponse), args.Error(1)
}

func (m *mockCourseAPIClient) SetUserState(ctx context.Context, req *pb.SetUserStateRequest, opts ...grpc.CallOption) (*pb.SetUserStateResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.SetUserStateResponse), args.Error(1)
}

func (m *mockCourseAPIClient) GetAllCourses(ctx context.Context, req *pb.GetAllCoursesRequest, opts ...grpc.CallOption) (*pb.GetAllCoursesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.GetAllCoursesResponse), args.Error(1)
}

// mockUserAPIClient is a mock implementation of pbUser.UserAPIClient for courses tests
type mockUserAPIClient struct {
	mock.Mock
}

func (m *mockUserAPIClient) SignUp(ctx context.Context, req *pbUser.SignUpRequest, opts ...grpc.CallOption) (*pbUser.SignUpResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.SignUpResponse), args.Error(1)
}

func (m *mockUserAPIClient) LogIn(ctx context.Context, req *pbUser.LogInRequest, opts ...grpc.CallOption) (*pbUser.LogInResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.LogInResponse), args.Error(1)
}

func (m *mockUserAPIClient) GetUserID(ctx context.Context, req *pbUser.GetUserIDRequest, opts ...grpc.CallOption) (*pbUser.GetUserIDResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.GetUserIDResponse), args.Error(1)
}

func (m *mockUserAPIClient) GetUsersByIDs(ctx context.Context, req *pbUser.GetUsersByIDsRequest, opts ...grpc.CallOption) (*pbUser.GetUsersByIDsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbUser.GetUsersByIDsResponse), args.Error(1)
}

// ============================================
// GetStudentsProgress Tests
// ============================================

func TestGetStudentsProgress(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		client.On("GetStudentsProgress", mock.Anything, mock.MatchedBy(func(req *pb.GetStudentsProgressRequest) bool {
			return req.CourseId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" && req.Pagination.First == 20
		})).Return(&pb.GetStudentsProgressResponse{
			Students: &pb.StudentsProgressConnection{
				Edges: []*pb.StudentsProgressConnection_Edge{
					{
						Node: &pb.StudentProgressSummary{
							UserId:                "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
							CurrentLessonId:       "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
							LessonsCompleted:      3,
							TotalLessons:          5,
							TotalCorrectAnswers:   15,
							TotalIncorrectAnswers: 2,
							EnrolledAt:            timestamppb.Now(),
							LastActivityAt:        timestamppb.Now(),
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: &pb.PageInfo{
					HasNextPage:     true,
					HasPreviousPage: false,
					StartCursor:     "cursor1",
					EndCursor:       "cursor1",
				},
			},
			TotalEnrolled: 50,
		}, nil)

		userClient.On("GetUsersByIDs", mock.Anything, mock.Anything).Return(&pbUser.GetUsersByIDsResponse{
			Users: []*pbUser.UserInfo{
				{Id: "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9", Username: "testuser", Nick: "Test User"},
			},
		}, nil)

		router := gin.New()
		router.GET("/admin/courses/:courseId/students/progress", func(ctx *gin.Context) {
			GetStudentsProgress(ctx, client, userClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/students/progress?first=20", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, float64(50), resp["total_enrolled"])

		edges := resp["edges"].([]interface{})
		assert.Len(t, edges, 1)

		edge := edges[0].(map[string]interface{})
		node := edge["node"].(map[string]interface{})
		assert.Equal(t, "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9", node["user_id"])
		assert.Equal(t, float64(3), node["lessons_completed"])
	})

	t.Run("success_with_detailed", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		client.On("GetStudentsProgress", mock.Anything, mock.MatchedBy(func(req *pb.GetStudentsProgressRequest) bool {
			return req.CourseId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" && req.Detailed == true
		})).Return(&pb.GetStudentsProgressResponse{
			Students: &pb.StudentsProgressConnection{
				Edges: []*pb.StudentsProgressConnection_Edge{
					{
						Node: &pb.StudentProgressSummary{
							UserId:           "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
							LessonsCompleted: 1,
							LessonDetails: []*pb.LessonProgressDetail{
								{
									LessonId:         "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
									IsCompleted:      true,
									CorrectAnswers:   10,
									IncorrectAnswers: 2,
								},
							},
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: &pb.PageInfo{},
			},
			TotalEnrolled: 1,
		}, nil)

		userClient.On("GetUsersByIDs", mock.Anything, mock.Anything).Return(&pbUser.GetUsersByIDsResponse{
			Users: []*pbUser.UserInfo{
				{Id: "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9", Username: "testuser", Nick: "Test User"},
			},
		}, nil)

		router := gin.New()
		router.GET("/admin/courses/:courseId/students/progress", func(ctx *gin.Context) {
			GetStudentsProgress(ctx, client, userClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/students/progress?detailed=true", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)

		edges := resp["edges"].([]interface{})
		edge := edges[0].(map[string]interface{})
		node := edge["node"].(map[string]interface{})
		lessonDetails := node["lesson_details"].([]interface{})
		assert.Len(t, lessonDetails, 1)
	})

	t.Run("failure_missing_course_id", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		router.GET("/admin/courses/:courseId/students/progress", func(ctx *gin.Context) {
			ctx.Params = gin.Params{} // Clear params
			GetStudentsProgress(ctx, client, userClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/test/students/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing course id")
	})

	t.Run("failure_grpc_error", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		client.On("GetStudentsProgress", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		router := gin.New()
		router.GET("/admin/courses/:courseId/students/progress", func(ctx *gin.Context) {
			GetStudentsProgress(ctx, client, userClient)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/students/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ============================================
// GetUserState Tests
// ============================================

func TestGetUserState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("GetUserState", mock.Anything, mock.MatchedBy(func(req *pb.GetUserStateRequest) bool {
			return req.CourseId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" &&
				req.UserId == "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9"
		})).Return(&pb.GetUserStateResponse{
			UserState: &pb.FullUserState{
				Id:              "8a5bc922-3def-4567-8901-2345abcdef67",
				UserId:          "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
				CourseId:        "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
				CurrentLessonId: "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
				StateJson:       `{"lessons":{"lesson1":{"is_completed":true}}}`,
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
			},
		}, nil)

		router := gin.New()
		router.GET("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			GetUserState(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp FullUserStateJSON
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9", resp.UserID)
		assert.Equal(t, "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72", resp.CourseID)
		assert.Contains(t, resp.StateJSON, "lesson1")
	})

	t.Run("failure_missing_ids", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			ctx.Params = gin.Params{} // Clear params
			GetUserState(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/test/users/test/state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing course id or user id")
	})

	t.Run("failure_not_found", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("GetUserState", mock.Anything, mock.Anything).Return(nil, &mockNotFoundError{})

		router := gin.New()
		router.GET("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			GetUserState(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ============================================
// SetUserState Tests
// ============================================

func TestSetUserState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("SetUserState", mock.Anything, mock.MatchedBy(func(req *pb.SetUserStateRequest) bool {
			return req.CourseId == "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72" &&
				req.UserId == "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9" &&
				req.CurrentLessonId == "334ddbf8-1acc-405b-86d8-49f0d1ca636c" &&
				req.StateJson == `{"lessons":{}}` &&
				req.SyncAfter == false
		})).Return(&pb.SetUserStateResponse{
			UserState: &pb.FullUserState{
				Id:              "8a5bc922-3def-4567-8901-2345abcdef67",
				UserId:          "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
				CourseId:        "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
				CurrentLessonId: "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
				StateJson:       `{"lessons":{}}`,
				CreatedAt:       timestamppb.Now(),
				UpdatedAt:       timestamppb.Now(),
			},
		}, nil)

		router := gin.New()
		router.PUT("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			SetUserState(ctx, client)
		})

		body, _ := json.Marshal(map[string]interface{}{
			"current_lesson_id": "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
			"state_json":        `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp FullUserStateJSON
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "334ddbf8-1acc-405b-86d8-49f0d1ca636c", resp.CurrentLessonID)
	})

	t.Run("success_with_sync", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("SetUserState", mock.Anything, mock.MatchedBy(func(req *pb.SetUserStateRequest) bool {
			return req.SyncAfter == true
		})).Return(&pb.SetUserStateResponse{
			UserState: &pb.FullUserState{
				Id:        "8a5bc922-3def-4567-8901-2345abcdef67",
				UserId:    "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
				CourseId:  "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
				StateJson: `{"lessons":{}}`,
			},
		}, nil)

		router := gin.New()
		router.PUT("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			SetUserState(ctx, client)
		})

		body, _ := json.Marshal(map[string]interface{}{
			"state_json": `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state?sync=true", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("failure_missing_ids", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.PUT("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			ctx.Params = gin.Params{} // Clear params
			SetUserState(ctx, client)
		})

		body, _ := json.Marshal(map[string]interface{}{
			"state_json": `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/test/users/test/state", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing course id or user id")
	})

	t.Run("failure_invalid_json_body", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.PUT("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			SetUserState(ctx, client)
		})

		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("failure_grpc_error", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("SetUserState", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		router := gin.New()
		router.PUT("/admin/courses/:courseId/users/:userId/state", func(ctx *gin.Context) {
			SetUserState(ctx, client)
		})

		body, _ := json.Marshal(map[string]interface{}{
			"state_json": `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ============================================
// Admin Middleware Integration Tests
// ============================================

func TestBackofficeRoutes_RequireAdmin(t *testing.T) {
	adminCfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "secret123",
	}

	t.Run("students_progress_without_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/students/progress", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("students_progress_with_valid_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		client.On("GetStudentsProgress", mock.Anything, mock.Anything).Return(&pb.GetStudentsProgressResponse{
			Students: &pb.StudentsProgressConnection{
				Edges:    []*pb.StudentsProgressConnection_Edge{},
				PageInfo: &pb.PageInfo{},
			},
			TotalEnrolled: 0,
		}, nil)

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/students/progress", nil)
		req.Header.Set("X-Admin-Token", "secret123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get_user_state_without_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("set_user_state_without_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		body, _ := json.Marshal(map[string]interface{}{
			"state_json": `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("set_user_state_with_invalid_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		body, _ := json.Marshal(map[string]interface{}{
			"state_json": `{"lessons":{}}`,
		})
		req := httptest.NewRequest(http.MethodPut, "/admin/courses/fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72/users/4e37a600-c29e-4d0f-af44-66f2cd8cc1c9/state", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Token", "wrong-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// mockNotFoundError simulates a "not found" error from gRPC
type mockNotFoundError struct{}

func (e *mockNotFoundError) Error() string {
	return "user progress not found"
}


func TestGetAllCourses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("GetAllCourses", mock.Anything, mock.MatchedBy(func(req *pb.GetAllCoursesRequest) bool {
			return req.Pagination.First == 20
		})).Return(&pb.GetAllCoursesResponse{
			Courses: &pb.CoursesConnection{
				Edges: []*pb.CoursesConnection_Edge{
					{
						Node: &pb.Course{
							Id:          "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
							Order:       1,
							Title:       "Go Mastery",
							Description: "Master the Go programming language",
							CreatedAt:   timestamppb.Now(),
						},
						Cursor: "cursor1",
					},
					{
						Node: &pb.Course{
							Id:          "1f30a72f-5d7a-48da-a5c2-42efece6972a",
							Order:       2,
							Title:       "Advanced Go",
							Description: "Advanced Go concepts",
							CreatedAt:   timestamppb.Now(),
						},
						Cursor: "cursor2",
					},
				},
				PageInfo: &pb.PageInfo{
					HasNextPage:     false,
					HasPreviousPage: false,
					StartCursor:     "cursor1",
					EndCursor:       "cursor2",
				},
			},
			TotalCount: 2,
		}, nil)

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		edges := response["edges"].([]interface{})
		assert.Len(t, edges, 2)

		totalCount := response["total_count"].(float64)
		assert.Equal(t, float64(2), totalCount)

		pageInfo := response["page_info"].(map[string]interface{})
		assert.False(t, pageInfo["has_next_page"].(bool))
	})

	t.Run("with_pagination", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		client.On("GetAllCourses", mock.Anything, mock.MatchedBy(func(req *pb.GetAllCoursesRequest) bool {
			return req.Pagination.First == 10 && req.Pagination.After == "some-cursor"
		})).Return(&pb.GetAllCoursesResponse{
			Courses: &pb.CoursesConnection{
				Edges:    []*pb.CoursesConnection_Edge{},
				PageInfo: &pb.PageInfo{},
			},
			TotalCount: 0,
		}, nil)

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?first=10&after=some-cursor", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("grpc_error", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		client.On("GetAllCourses", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("non_numeric_first", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?first=abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("negative_first", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?first=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("negative_last", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?last=-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("first_above_max", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?first=1001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("last_max_int64", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?last=9223372036854775807", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("last_above_max", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?last=1001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("first_max_int64", func(t *testing.T) {
		client := &mockCourseAPIClient{}

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?first=9223372036854775807", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		client.AssertNotCalled(t, "GetAllCourses", mock.Anything, mock.Anything)
	})

	t.Run("invalid_cursor", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		client.On("GetAllCourses", mock.Anything, mock.Anything).
			Return(nil, status.Error(codes.InvalidArgument, "invalid cursor"))

		router := gin.New()
		router.GET("/admin/courses", func(ctx *gin.Context) {
			GetAllCourses(ctx, client)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin/courses?after=garbage", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetAllCoursesRouteRequiresAdmin(t *testing.T) {
	adminCfg := AdminConfig{
		HeaderName:   "X-Admin-Token",
		HeaderSecret: "test-secret",
	}

	t.Run("without_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("with_invalid_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses", nil)
		req.Header.Set("X-Admin-Token", "wrong-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("with_valid_admin_header", func(t *testing.T) {
		client := &mockCourseAPIClient{}
		userClient := &mockUserAPIClient{}

		client.On("GetAllCourses", mock.Anything, mock.Anything).Return(&pb.GetAllCoursesResponse{
			Courses: &pb.CoursesConnection{
				Edges:    []*pb.CoursesConnection_Edge{},
				PageInfo: &pb.PageInfo{},
			},
			TotalCount: 0,
		}, nil)

		router := gin.New()
		RegisterCoursesRoutes(router.Group("/"), client, userClient, adminCfg)

		req := httptest.NewRequest(http.MethodGet, "/admin/courses", nil)
		req.Header.Set("X-Admin-Token", "test-secret")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
