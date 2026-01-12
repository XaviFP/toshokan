package grpc

import (
	"context"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/tilinna/clock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/XaviFP/toshokan/common/pagination"
	pb "github.com/XaviFP/toshokan/course/api/proto/v1"
	course "github.com/XaviFP/toshokan/course/internal/course"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

// Server implements the courses gRPC server
type Server struct {
	pb.UnimplementedCourseAPIServer
	GRPCAddr       string
	GRPCTransport  string
	Repository     course.Repository
	Enroller       course.Enroller
	DeckClient     pbDeck.DecksAPIClient
	LessonsBrowser course.LessonsBrowser
	CoursesBrowser course.CoursesBrowser
	Answerer       course.Answerer
	Clock          clock.Clock

	grpcServer *grpc.Server
}

// Start starts the gRPC server
func (s *Server) Start() error {
	log.Printf("Starting Course gRPC server on: %s", s.GRPCAddr)

	// lets print all operations
	s.grpcServer = grpc.NewServer()
	pb.RegisterCourseAPIServer(s.grpcServer, s)

	listener, err := net.Listen(s.GRPCTransport, s.GRPCAddr)
	if err != nil {
		return errors.Annotatef(err, "failed to listen on %s", s.GRPCAddr)
	}

	if err := s.grpcServer.Serve(listener); err != nil {
		return errors.Annotatef(err, "failed to serve on %s", s.GRPCAddr)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func toProtoLessonState(lessonStateMap course.LessonState) map[string]*pb.LessonState {
	pbLessonState := make(map[string]*pb.LessonState)
	for lessonKey, lesson := range lessonStateMap {

		pbDecks := make(map[string]*pb.DeckState)
		for deckKey, deck := range lesson.Decks {

			pbCards := make(map[string]*pb.CardState)
			for cardKey, card := range deck.Cards {
				pbCard := &pb.CardState{
					CorrectAnswers:   int32(card.CorrectAnswers),
					IncorrectAnswers: int32(card.IncorrectAnswers),
					IsCompleted:      card.IsCompleted,
				}
				if card.CompletedAt != nil {
					pbCard.CompletedAt = timestamppb.New(*card.CompletedAt)
				}
				pbCards[cardKey] = pbCard
			}

			pbDeck := &pb.DeckState{
				Cards:       pbCards,
				IsCompleted: deck.IsCompleted,
			}
			if deck.CompletedAt != nil {
				pbDeck.CompletedAt = timestamppb.New(*deck.CompletedAt)
			}
			pbDecks[deckKey] = pbDeck
		}

		pbLesson := &pb.LessonState{
			Decks:       pbDecks,
			IsCompleted: lesson.IsCompleted,
		}
		if lesson.CompletedAt != nil {
			pbLesson.CompletedAt = timestamppb.New(*lesson.CompletedAt)
		}
		pbLessonState[lessonKey] = pbLesson
	}
	return pbLessonState
}

// GetCourse retrieves a course by ID
func (s *Server) GetCourse(ctx context.Context, req *pb.GetCourseRequest) (*pb.GetCourseResponse, error) {
	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("GetCourse: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	course, err := s.Repository.GetCourse(ctx, courseID)
	if err != nil {
		slog.Error("GetCourse: failed to get course from repository", "error", err, "courseId", courseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.GetCourseResponse{
		Course: &pb.Course{
			Id:          course.ID.String(),
			Title:       course.Title,
			Description: course.Description,
			CreatedAt:   toProtoTimestamp(course.CreatedAt),
			EditedAt:    toProtoTimestampPtr(course.EditedAt),
			DeletedAt:   toProtoTimestampPtr(course.DeletedAt),
		},
	}, nil
}

// GetLesson retrieves a lesson by ID
func (s *Server) GetLesson(ctx context.Context, req *pb.GetLessonRequest) (*pb.GetLessonResponse, error) {
	lessonID, err := uuid.Parse(req.LessonId)
	if err != nil {
		slog.Error("GetLesson: failed to parse lesson ID", "error", err, "lessonId", req.LessonId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	lesson, err := s.Repository.GetLesson(ctx, lessonID)
	if err != nil {
		slog.Error("GetLesson: failed to get lesson from repository", "error", err, "lessonId", lessonID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.GetLessonResponse{
		Lesson: &pb.Lesson{
			Id:          lesson.ID.String(),
			CourseId:    lesson.CourseID.String(),
			Order:       int64(lesson.Order),
			Title:       lesson.Title,
			Description: lesson.Description,
			Body:        lesson.Body,
			CreatedAt:   toProtoTimestamp(lesson.CreatedAt),
			EditedAt:    toProtoTimestampPtr(lesson.EditedAt),
			DeletedAt:   toProtoTimestampPtr(lesson.DeletedAt),
		},
	}, nil
}

// GetLessons retrieves lessons for a course with pagination
func (s *Server) GetLessons(ctx context.Context, req *pb.GetLessonsRequest) (*pb.GetLessonsResponse, error) {
	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("GetLessons: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	p := pagination.NewOldestFirstPagination(
		pagination.WithFirst(int(req.Pagination.First)),
		pagination.WithLast(int(req.Pagination.Last)),
		pagination.WithAfter(pagination.Cursor(req.Pagination.After)),
		pagination.WithBefore(pagination.Cursor(req.Pagination.Before)),
	)

	// Public endpoint - no user context
	result, err := s.LessonsBrowser.Browse(ctx, courseID, p, course.BrowseOptions{})
	if err != nil {
		slog.Error("GetLessons: failed to browse lessons", "error", err, "courseId", courseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	resp := &pb.GetLessonsResponse{
		Lessons: &pb.LessonsConnection{
			PageInfo: &pb.PageInfo{
				HasPreviousPage: result.PublicLessons.PageInfo.HasPreviousPage,
				HasNextPage:     result.PublicLessons.PageInfo.HasNextPage,
				StartCursor:     result.PublicLessons.PageInfo.StartCursor.String(),
				EndCursor:       result.PublicLessons.PageInfo.EndCursor.String(),
			},
		},
	}

	for _, edge := range result.PublicLessons.Edges {
		resp.Lessons.Edges = append(resp.Lessons.Edges, &pb.LessonsConnection_Edge{
			Node:   lessonToProto(&edge.Lesson),
			Cursor: string(edge.Cursor),
		})
	}

	return resp, nil
}

// GetFocusedLessons retrieves lessons around current lesson
func (s *Server) GetFocusedLessons(ctx context.Context, req *pb.GetFocusedLessonsRequest) (*pb.GetFocusedLessonsResponse, error) {
	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("GetFocusedLessons: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("GetFocusedLessons: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	p := pagination.NewOldestFirstPagination(
		pagination.WithFirst(int(req.Pagination.First)),
		pagination.WithLast(int(req.Pagination.Last)),
		pagination.WithAfter(pagination.Cursor(req.Pagination.After)),
		pagination.WithBefore(pagination.Cursor(req.Pagination.Before)),
	)

	// Authenticated endpoint - with user context
	result, err := s.LessonsBrowser.Browse(ctx, courseID, p, course.BrowseOptions{
		UserID: &userID,
	})
	if err != nil {
		slog.Error("GetFocusedLessons: failed to browse lessons", "error", err, "courseId", courseID.String(), "userId", userID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	resp := &pb.GetFocusedLessonsResponse{
		Lessons: &pb.LessonsWithProgressConnection{
			PageInfo: &pb.PageInfo{
				HasPreviousPage: result.ProgressLessons.PageInfo.HasPreviousPage,
				HasNextPage:     result.ProgressLessons.PageInfo.HasNextPage,
				StartCursor:     result.ProgressLessons.PageInfo.StartCursor.String(),
				EndCursor:       result.ProgressLessons.PageInfo.EndCursor.String(),
			},
		},
	}

	for _, edge := range result.ProgressLessons.Edges {
		resp.Lessons.Edges = append(resp.Lessons.Edges, &pb.LessonsWithProgressConnection_Edge{
			Node: &pb.LessonWithProgress{
				Lesson:      lessonToProto(&edge.Lesson.Lesson),
				IsCompleted: edge.Lesson.IsCompleted,
				IsCurrent:   edge.Lesson.IsCurrent,
			},
			Cursor: string(edge.Cursor),
		})
	}

	return resp, nil
}

// GetEnrolledCourses retrieves courses a user is enrolled in with progress
func (s *Server) GetEnrolledCourses(ctx context.Context, req *pb.GetEnrolledCoursesRequest) (*pb.GetEnrolledCoursesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("GetEnrolledCourses: failed to parse user ID", "error", err, "userId", req.UserId, "userIdLen", len(req.UserId), "stack", errors.ErrorStack(err))
		return nil, errors.Annotatef(err, "invalid user_id: %q", req.UserId)
	}

	p := pagination.NewOldestFirstPagination(
		pagination.WithFirst(int(req.Pagination.First)),
		pagination.WithLast(int(req.Pagination.Last)),
		pagination.WithAfter(pagination.Cursor(req.Pagination.After)),
		pagination.WithBefore(pagination.Cursor(req.Pagination.Before)),
	)

	conn, err := s.CoursesBrowser.BrowseEnrolled(ctx, userID, p)
	if err != nil {
		slog.Error("GetEnrolledCourses: failed to browse enrolled courses", "error", err, "userId", userID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	resp := &pb.GetEnrolledCoursesResponse{
		Courses: &pb.CoursesWithProgressConnection{
			PageInfo: &pb.PageInfo{
				HasPreviousPage: conn.PageInfo.HasPreviousPage,
				HasNextPage:     conn.PageInfo.HasNextPage,
				StartCursor:     conn.PageInfo.StartCursor.String(),
				EndCursor:       conn.PageInfo.EndCursor.String(),
			},
		},
	}

	for _, edge := range conn.Edges {
		resp.Courses.Edges = append(resp.Courses.Edges, &pb.CoursesWithProgressConnection_Edge{
			Node: &pb.CourseWithProgress{
				Course:          courseToProto(&edge.Course.Course),
				CurrentLessonId: edge.Course.CurrentLessonID,
			},
			Cursor: string(edge.Cursor),
		})
	}

	return resp, nil
}

func courseToProto(c *course.Course) *pb.Course {
	return &pb.Course{
		Id:          c.ID.String(),
		Order:       c.Order,
		Title:       c.Title,
		Description: c.Description,
		CreatedAt:   toProtoTimestamp(c.CreatedAt),
		EditedAt:    toProtoTimestampPtr(c.EditedAt),
		DeletedAt:   toProtoTimestampPtr(c.DeletedAt),
	}
}

func lessonToProto(l *course.Lesson) *pb.Lesson {
	return &pb.Lesson{
		Id:          l.ID.String(),
		CourseId:    l.CourseID.String(),
		Order:       int64(l.Order),
		Title:       l.Title,
		Description: l.Description,
		Body:        l.Body,
		CreatedAt:   timestamppb.New(l.CreatedAt),
	}
}

// EnrollUser enrolls a user in a course
func (s *Server) EnrollUser(ctx context.Context, req *pb.EnrollUserRequest) (*pb.EnrollUserResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("EnrollUser: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("EnrollUser: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	_, err = s.Enroller.Enroll(ctx, userID, courseID)
	if err != nil {
		slog.Error("EnrollUser: failed to enroll user", "error", err, "userId", userID.String(), "courseId", courseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.EnrollUserResponse{Success: true}, nil
}

// GetUserProgress retrieves user progress
func (s *Server) GetUserProgress(ctx context.Context, req *pb.GetUserProgressRequest) (*pb.GetUserProgressResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("GetUserProgress: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("GetUserProgress: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	progress, err := s.Repository.GetUserCourseProgress(ctx, userID, courseID)
	if err != nil {
		slog.Error("GetUserProgress: failed to get user course progress", "error", err, "userId", userID.String(), "courseId", courseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	currentLessonID := ""
	if progress.CurrentLessonID != uuid.Nil {
		currentLessonID = progress.CurrentLessonID.String()
	}

	return &pb.GetUserProgressResponse{
		Progress: &pb.UserCourseProgress{
			Id:              progress.ID.String(),
			UserId:          progress.UserID.String(),
			CourseId:        progress.CourseID.String(),
			CurrentLessonId: currentLessonID,
			CreatedAt:       toProtoTimestamp(progress.CreatedAt),
			UpdatedAt:       toProtoTimestamp(progress.UpdatedAt),
		},
	}, nil
}

// CreateCourse creates a new course
func (s *Server) CreateCourse(ctx context.Context, req *pb.CreateCourseRequest) (*pb.CreateCourseResponse, error) {
	if req.Title == "" {
		slog.Error("CreateCourse: missing title", "error", course.ErrNoTitle)
		return nil, errors.Trace(course.ErrNoTitle)
	}
	if req.Description == "" {
		slog.Error("CreateCourse: missing description", "error", course.ErrNoDescription)
		return nil, errors.Trace(course.ErrNoDescription)
	}

	course := course.Course{
		ID:          uuid.New(),
		Order:       req.Order,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	err := s.Repository.StoreCourse(ctx, course)
	if err != nil {
		slog.Error("CreateCourse: failed to store course", "error", err, "courseId", course.ID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.CreateCourseResponse{
		Course: &pb.Course{
			Id:          course.ID.String(),
			Order:       course.Order,
			Title:       course.Title,
			Description: course.Description,
			CreatedAt:   toProtoTimestamp(course.CreatedAt),
		},
	}, nil
}

// CreateLesson creates a new lesson
func (s *Server) CreateLesson(ctx context.Context, req *pb.CreateLessonRequest) (*pb.CreateLessonResponse, error) {
	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("CreateLesson: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	if req.Title == "" {
		slog.Error("CreateLesson: missing title", "error", course.ErrNoTitle)
		return nil, errors.Trace(course.ErrNoTitle)
	}
	if req.Description == "" {
		slog.Error("CreateLesson: missing description", "error", course.ErrNoDescription)
		return nil, errors.Trace(course.ErrNoDescription)
	}

	// Parse body for deck references
	deckIDs := course.ParseDeckReferences(req.Body)

	// Require at least one deck
	if len(deckIDs) == 0 {
		err := errors.New("lesson must reference at least one deck in the body using ![deck](uuid) format")
		slog.Error("CreateLesson: no deck references found", "error", err)
		return nil, errors.Trace(err)
	}

	// Validate that all decks exist
	for _, deckID := range deckIDs {
		deckReq := &pbDeck.GetDeckRequest{DeckId: deckID.String()}
		_, err := s.DeckClient.GetDeck(ctx, deckReq)
		if err != nil {
			slog.Error("CreateLesson: deck does not exist", "error", err, "deckId", deckID.String(), "stack", errors.ErrorStack(err))
			return nil, errors.Annotatef(err, "deck %s does not exist", deckID.String())
		}
	}

	lesson := course.Lesson{
		ID:          uuid.New(),
		CourseID:    courseID,
		Order:       int(req.Order),
		Title:       req.Title,
		Description: req.Description,
		Body:        req.Body,
		CreatedAt:   time.Now(),
	}

	err = s.Repository.StoreLesson(ctx, lesson)
	if err != nil {
		slog.Error("CreateLesson: failed to store lesson", "error", err, "lessonId", lesson.ID.String(), "courseId", lesson.CourseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.CreateLessonResponse{
		Lesson: &pb.Lesson{
			Id:          lesson.ID.String(),
			CourseId:    lesson.CourseID.String(),
			Order:       int64(lesson.Order),
			Title:       lesson.Title,
			Description: lesson.Description,
			Body:        lesson.Body,
			CreatedAt:   toProtoTimestamp(lesson.CreatedAt),
		},
	}, nil
}

// GetLessonState returns the completion state of a lesson and its decks
func (s *Server) GetLessonState(ctx context.Context, req *pb.GetLessonStateRequest) (*pb.GetLessonStateResponse, error) {
	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("GetLessonState: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	lessonID, err := uuid.Parse(req.LessonId)
	if err != nil {
		slog.Error("GetLessonState: failed to parse lesson ID", "error", err, "lessonId", req.LessonId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("GetLessonState: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	// Get user progress
	progress, err := s.Repository.GetUserCourseProgress(ctx, userID, courseID)
	if err != nil {
		slog.Error("GetLessonState: failed to get user course progress", "error", err, "userId", userID.String(), "courseId", courseID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	// Get lesson state from progress
	lessonStateMap := progress.State.GetLessonState(lessonID.String())

	// Convert to protobuf response
	pbLessonState := toProtoLessonState(lessonStateMap)

	return &pb.GetLessonStateResponse{LessonState: pbLessonState}, nil
}

// AnswerCards handles card answer submissions
func (s *Server) AnswerCards(ctx context.Context, req *pb.AnswerCardsRequest) (*pb.AnswerCardsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("AnswerCards: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	courseID, err := uuid.Parse(req.CourseId)
	if err != nil {
		slog.Error("AnswerCards: failed to parse course ID", "error", err, "courseId", req.CourseId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	lessonID, err := uuid.Parse(req.LessonId)
	if err != nil {
		slog.Error("AnswerCards: failed to parse lesson ID", "error", err, "lessonId", req.LessonId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	deckID, err := uuid.Parse(req.DeckId)
	if err != nil {
		slog.Error("AnswerCards: failed to parse deck ID", "error", err, "deckId", req.DeckId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	cardAnswers := make([]course.CardAnswer, len(req.CardAnswers))
	for i, ca := range req.CardAnswers {
		cardID, err := uuid.Parse(ca.CardId)
		if err != nil {
			slog.Error("AnswerCards: failed to parse card ID", "error", err, "cardId", ca.CardId, "index", i, "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}
		answerID, err := uuid.Parse(ca.AnswerId)
		if err != nil {
			slog.Error("AnswerCards: failed to parse answer ID", "error", err, "answerId", ca.AnswerId, "cardId", cardID.String(), "index", i, "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}
		cardAnswers[i] = course.CardAnswer{
			CardID:   cardID,
			AnswerID: answerID,
		}
	}

	err = s.Answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)
	if err != nil {
		slog.Error("AnswerCards: failed to answer cards", "error", err, "userId", userID.String(), "courseId", courseID.String(), "lessonId", lessonID.String(), "deckId", deckID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.AnswerCardsResponse{
		Success: true,
	}, nil
}
