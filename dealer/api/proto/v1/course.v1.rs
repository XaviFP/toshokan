#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Course {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(int64, tag = "2")]
    pub order: i64,
    #[prost(string, tag = "3")]
    pub title: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub description: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "5")]
    pub created_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "6")]
    pub updated_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "7")]
    pub deleted_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Lesson {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(int64, tag = "3")]
    pub order: i64,
    #[prost(string, tag = "4")]
    pub title: ::prost::alloc::string::String,
    #[prost(string, tag = "5")]
    pub description: ::prost::alloc::string::String,
    #[prost(string, tag = "6")]
    pub body: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "7")]
    pub created_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "8")]
    pub updated_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "9")]
    pub deleted_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct UserCourseProgress {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub current_lesson_id: ::prost::alloc::string::String,
    /// JSON string
    #[prost(string, tag = "5")]
    pub state: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "6")]
    pub created_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "7")]
    pub updated_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PageInfo {
    #[prost(bool, tag = "1")]
    pub has_previous_page: bool,
    #[prost(bool, tag = "2")]
    pub has_next_page: bool,
    #[prost(string, tag = "3")]
    pub start_cursor: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub end_cursor: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pagination {
    #[prost(int64, tag = "1")]
    pub last: i64,
    #[prost(int64, tag = "2")]
    pub first: i64,
    #[prost(string, tag = "3")]
    pub after: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub before: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LessonWithProgress {
    #[prost(message, optional, tag = "1")]
    pub lesson: ::core::option::Option<Lesson>,
    #[prost(bool, tag = "2")]
    pub is_completed: bool,
    #[prost(bool, tag = "3")]
    pub is_current: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LessonsConnection {
    #[prost(message, repeated, tag = "1")]
    pub edges: ::prost::alloc::vec::Vec<lessons_connection::Edge>,
    #[prost(message, optional, tag = "2")]
    pub page_info: ::core::option::Option<PageInfo>,
}
/// Nested message and enum types in `LessonsConnection`.
pub mod lessons_connection {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Edge {
        #[prost(message, optional, tag = "1")]
        pub node: ::core::option::Option<super::Lesson>,
        #[prost(string, tag = "2")]
        pub cursor: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LessonsWithProgressConnection {
    #[prost(message, repeated, tag = "1")]
    pub edges: ::prost::alloc::vec::Vec<lessons_with_progress_connection::Edge>,
    #[prost(message, optional, tag = "2")]
    pub page_info: ::core::option::Option<PageInfo>,
}
/// Nested message and enum types in `LessonsWithProgressConnection`.
pub mod lessons_with_progress_connection {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Edge {
        #[prost(message, optional, tag = "1")]
        pub node: ::core::option::Option<super::LessonWithProgress>,
        #[prost(string, tag = "2")]
        pub cursor: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CourseWithProgress {
    #[prost(message, optional, tag = "1")]
    pub course: ::core::option::Option<Course>,
    #[prost(string, tag = "2")]
    pub current_lesson_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CoursesWithProgressConnection {
    #[prost(message, repeated, tag = "1")]
    pub edges: ::prost::alloc::vec::Vec<courses_with_progress_connection::Edge>,
    #[prost(message, optional, tag = "2")]
    pub page_info: ::core::option::Option<PageInfo>,
}
/// Nested message and enum types in `CoursesWithProgressConnection`.
pub mod courses_with_progress_connection {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Edge {
        #[prost(message, optional, tag = "1")]
        pub node: ::core::option::Option<super::CourseWithProgress>,
        #[prost(string, tag = "2")]
        pub cursor: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetCourseRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetCourseResponse {
    #[prost(message, optional, tag = "1")]
    pub course: ::core::option::Option<Course>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonRequest {
    #[prost(string, tag = "1")]
    pub lesson_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonResponse {
    #[prost(message, optional, tag = "1")]
    pub lesson: ::core::option::Option<Lesson>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetCourseDeckIdsRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetCourseDeckIdsResponse {
    #[prost(string, repeated, tag = "1")]
    pub deck_ids: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonDeckIdsRequest {
    #[prost(string, tag = "1")]
    pub lesson_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonDeckIdsResponse {
    #[prost(string, repeated, tag = "1")]
    pub deck_ids: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonsRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    /// No user_id - this is public endpoint, returns Lesson without progress
    #[prost(message, optional, tag = "2")]
    pub pagination: ::core::option::Option<Pagination>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonsResponse {
    /// Public - bare lessons
    #[prost(message, optional, tag = "1")]
    pub lessons: ::core::option::Option<LessonsConnection>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetFocusedLessonsRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    /// User context provided - returns LessonWithProgress
    #[prost(string, tag = "2")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "3")]
    pub pagination: ::core::option::Option<Pagination>,
    /// If true, omit lesson body from response for faster pagination
    #[prost(bool, tag = "4")]
    pub bodyless: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetFocusedLessonsResponse {
    /// Authenticated - with progress
    #[prost(message, optional, tag = "1")]
    pub lessons: ::core::option::Option<LessonsWithProgressConnection>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetEnrolledCoursesRequest {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "2")]
    pub pagination: ::core::option::Option<Pagination>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetEnrolledCoursesResponse {
    #[prost(message, optional, tag = "1")]
    pub courses: ::core::option::Option<CoursesWithProgressConnection>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EnrollUserRequest {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub course_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct EnrollUserResponse {
    #[prost(bool, tag = "1")]
    pub success: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetUserProgressRequest {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub course_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetUserProgressResponse {
    #[prost(message, optional, tag = "1")]
    pub progress: ::core::option::Option<UserCourseProgress>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct SyncStateRequest {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub course_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct SyncStateResponse {}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CardAnswer {
    #[prost(string, tag = "1")]
    pub card_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub answer_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AnswerCardsRequest {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub lesson_id: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub deck_id: ::prost::alloc::string::String,
    #[prost(message, repeated, tag = "5")]
    pub card_answers: ::prost::alloc::vec::Vec<CardAnswer>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct AnswerCardsResponse {
    #[prost(bool, tag = "1")]
    pub success: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateCourseRequest {
    #[prost(int64, tag = "1")]
    pub order: i64,
    #[prost(string, tag = "2")]
    pub title: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub description: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateCourseResponse {
    #[prost(message, optional, tag = "1")]
    pub course: ::core::option::Option<Course>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateLessonRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(int64, tag = "2")]
    pub order: i64,
    #[prost(string, tag = "3")]
    pub title: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub description: ::prost::alloc::string::String,
    #[prost(string, tag = "5")]
    pub body: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CreateLessonResponse {
    #[prost(message, optional, tag = "1")]
    pub lesson: ::core::option::Option<Lesson>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CardState {
    #[prost(int32, tag = "1")]
    pub correct_answers: i32,
    #[prost(int32, tag = "2")]
    pub incorrect_answers: i32,
    #[prost(bool, tag = "3")]
    pub is_completed: bool,
    #[prost(message, optional, tag = "4")]
    pub completed_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct DeckState {
    #[prost(map = "string, message", tag = "1")]
    pub cards: ::std::collections::HashMap<::prost::alloc::string::String, CardState>,
    #[prost(bool, tag = "2")]
    pub is_completed: bool,
    #[prost(message, optional, tag = "3")]
    pub completed_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LessonState {
    #[prost(map = "string, message", tag = "1")]
    pub decks: ::std::collections::HashMap<::prost::alloc::string::String, DeckState>,
    #[prost(bool, tag = "2")]
    pub is_completed: bool,
    #[prost(message, optional, tag = "3")]
    pub completed_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonStateRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub lesson_id: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub user_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetLessonStateResponse {
    #[prost(map = "string, message", tag = "1")]
    pub lesson_state: ::std::collections::HashMap<
        ::prost::alloc::string::String,
        LessonState,
    >,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct UpdateCourseRequest {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(int64, optional, tag = "2")]
    pub order: ::core::option::Option<i64>,
    #[prost(string, optional, tag = "3")]
    pub title: ::core::option::Option<::prost::alloc::string::String>,
    #[prost(string, optional, tag = "4")]
    pub description: ::core::option::Option<::prost::alloc::string::String>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct UpdateCourseResponse {
    #[prost(message, optional, tag = "1")]
    pub course: ::core::option::Option<Course>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct UpdateLessonRequest {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(int64, optional, tag = "2")]
    pub order: ::core::option::Option<i64>,
    #[prost(string, optional, tag = "3")]
    pub title: ::core::option::Option<::prost::alloc::string::String>,
    #[prost(string, optional, tag = "4")]
    pub description: ::core::option::Option<::prost::alloc::string::String>,
    #[prost(string, optional, tag = "5")]
    pub body: ::core::option::Option<::prost::alloc::string::String>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct UpdateLessonResponse {
    #[prost(message, optional, tag = "1")]
    pub lesson: ::core::option::Option<Lesson>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct StudentProgressSummary {
    #[prost(string, tag = "1")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub current_lesson_id: ::prost::alloc::string::String,
    #[prost(int32, tag = "3")]
    pub lessons_completed: i32,
    #[prost(int32, tag = "4")]
    pub total_lessons: i32,
    #[prost(int32, tag = "5")]
    pub total_correct_answers: i32,
    #[prost(int32, tag = "6")]
    pub total_incorrect_answers: i32,
    #[prost(message, optional, tag = "7")]
    pub enrolled_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "8")]
    pub last_activity_at: ::core::option::Option<::prost_types::Timestamp>,
    /// Optional detailed breakdown per lesson (when detailed=true)
    #[prost(message, repeated, tag = "9")]
    pub lesson_details: ::prost::alloc::vec::Vec<LessonProgressDetail>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct LessonProgressDetail {
    #[prost(string, tag = "1")]
    pub lesson_id: ::prost::alloc::string::String,
    #[prost(bool, tag = "2")]
    pub is_completed: bool,
    #[prost(message, optional, tag = "3")]
    pub completed_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(int32, tag = "4")]
    pub correct_answers: i32,
    #[prost(int32, tag = "5")]
    pub incorrect_answers: i32,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct StudentsProgressConnection {
    #[prost(message, repeated, tag = "1")]
    pub edges: ::prost::alloc::vec::Vec<students_progress_connection::Edge>,
    #[prost(message, optional, tag = "2")]
    pub page_info: ::core::option::Option<PageInfo>,
}
/// Nested message and enum types in `StudentsProgressConnection`.
pub mod students_progress_connection {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Edge {
        #[prost(message, optional, tag = "1")]
        pub node: ::core::option::Option<super::StudentProgressSummary>,
        #[prost(string, tag = "2")]
        pub cursor: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetStudentsProgressRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "2")]
    pub pagination: ::core::option::Option<Pagination>,
    /// If true, include per-lesson breakdown
    #[prost(bool, tag = "3")]
    pub detailed: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetStudentsProgressResponse {
    #[prost(message, optional, tag = "1")]
    pub students: ::core::option::Option<StudentsProgressConnection>,
    /// Total number of enrolled students (for UI)
    #[prost(int32, tag = "2")]
    pub total_enrolled: i32,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct FullUserState {
    #[prost(string, tag = "1")]
    pub id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub user_id: ::prost::alloc::string::String,
    #[prost(string, tag = "3")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "4")]
    pub current_lesson_id: ::prost::alloc::string::String,
    /// Full ProgressState as JSON
    #[prost(string, tag = "5")]
    pub state_json: ::prost::alloc::string::String,
    #[prost(message, optional, tag = "6")]
    pub created_at: ::core::option::Option<::prost_types::Timestamp>,
    #[prost(message, optional, tag = "7")]
    pub updated_at: ::core::option::Option<::prost_types::Timestamp>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetUserStateRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub user_id: ::prost::alloc::string::String,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetUserStateResponse {
    #[prost(message, optional, tag = "1")]
    pub user_state: ::core::option::Option<FullUserState>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct SetUserStateRequest {
    #[prost(string, tag = "1")]
    pub course_id: ::prost::alloc::string::String,
    #[prost(string, tag = "2")]
    pub user_id: ::prost::alloc::string::String,
    /// Optional: update current lesson
    #[prost(string, tag = "3")]
    pub current_lesson_id: ::prost::alloc::string::String,
    /// Full ProgressState as JSON
    #[prost(string, tag = "4")]
    pub state_json: ::prost::alloc::string::String,
    /// If true, run sync after setting state
    #[prost(bool, tag = "5")]
    pub sync_after: bool,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct SetUserStateResponse {
    /// Return the updated state
    #[prost(message, optional, tag = "1")]
    pub user_state: ::core::option::Option<FullUserState>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct CoursesConnection {
    #[prost(message, repeated, tag = "1")]
    pub edges: ::prost::alloc::vec::Vec<courses_connection::Edge>,
    #[prost(message, optional, tag = "2")]
    pub page_info: ::core::option::Option<PageInfo>,
}
/// Nested message and enum types in `CoursesConnection`.
pub mod courses_connection {
    #[allow(clippy::derive_partial_eq_without_eq)]
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Edge {
        #[prost(message, optional, tag = "1")]
        pub node: ::core::option::Option<super::Course>,
        #[prost(string, tag = "2")]
        pub cursor: ::prost::alloc::string::String,
    }
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetAllCoursesRequest {
    #[prost(message, optional, tag = "1")]
    pub pagination: ::core::option::Option<Pagination>,
}
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct GetAllCoursesResponse {
    #[prost(message, optional, tag = "1")]
    pub courses: ::core::option::Option<CoursesConnection>,
    #[prost(int32, tag = "2")]
    pub total_count: i32,
}
/// Generated client implementations.
pub mod course_api_client {
    #![allow(unused_variables, dead_code, missing_docs, clippy::let_unit_value)]
    use tonic::codegen::*;
    use tonic::codegen::http::Uri;
    #[derive(Debug, Clone)]
    pub struct CourseApiClient<T> {
        inner: tonic::client::Grpc<T>,
    }
    impl CourseApiClient<tonic::transport::Channel> {
        /// Attempt to create a new client by connecting to a given endpoint.
        pub async fn connect<D>(dst: D) -> Result<Self, tonic::transport::Error>
        where
            D: std::convert::TryInto<tonic::transport::Endpoint>,
            D::Error: Into<StdError>,
        {
            let conn = tonic::transport::Endpoint::new(dst)?.connect().await?;
            Ok(Self::new(conn))
        }
    }
    impl<T> CourseApiClient<T>
    where
        T: tonic::client::GrpcService<tonic::body::BoxBody>,
        T::Error: Into<StdError>,
        T::ResponseBody: Body<Data = Bytes> + Send + 'static,
        <T::ResponseBody as Body>::Error: Into<StdError> + Send,
    {
        pub fn new(inner: T) -> Self {
            let inner = tonic::client::Grpc::new(inner);
            Self { inner }
        }
        pub fn with_origin(inner: T, origin: Uri) -> Self {
            let inner = tonic::client::Grpc::with_origin(inner, origin);
            Self { inner }
        }
        pub fn with_interceptor<F>(
            inner: T,
            interceptor: F,
        ) -> CourseApiClient<InterceptedService<T, F>>
        where
            F: tonic::service::Interceptor,
            T::ResponseBody: Default,
            T: tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
                Response = http::Response<
                    <T as tonic::client::GrpcService<tonic::body::BoxBody>>::ResponseBody,
                >,
            >,
            <T as tonic::codegen::Service<
                http::Request<tonic::body::BoxBody>,
            >>::Error: Into<StdError> + Send + Sync,
        {
            CourseApiClient::new(InterceptedService::new(inner, interceptor))
        }
        /// Compress requests with the given encoding.
        ///
        /// This requires the server to support it otherwise it might respond with an
        /// error.
        #[must_use]
        pub fn send_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.send_compressed(encoding);
            self
        }
        /// Enable decompressing responses.
        #[must_use]
        pub fn accept_compressed(mut self, encoding: CompressionEncoding) -> Self {
            self.inner = self.inner.accept_compressed(encoding);
            self
        }
        pub async fn get_course(
            &mut self,
            request: impl tonic::IntoRequest<super::GetCourseRequest>,
        ) -> Result<tonic::Response<super::GetCourseResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetCourse",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_lesson(
            &mut self,
            request: impl tonic::IntoRequest<super::GetLessonRequest>,
        ) -> Result<tonic::Response<super::GetLessonResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetLesson",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_lessons(
            &mut self,
            request: impl tonic::IntoRequest<super::GetLessonsRequest>,
        ) -> Result<tonic::Response<super::GetLessonsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetLessons",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_focused_lessons(
            &mut self,
            request: impl tonic::IntoRequest<super::GetFocusedLessonsRequest>,
        ) -> Result<tonic::Response<super::GetFocusedLessonsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetFocusedLessons",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_enrolled_courses(
            &mut self,
            request: impl tonic::IntoRequest<super::GetEnrolledCoursesRequest>,
        ) -> Result<tonic::Response<super::GetEnrolledCoursesResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetEnrolledCourses",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn enroll_user(
            &mut self,
            request: impl tonic::IntoRequest<super::EnrollUserRequest>,
        ) -> Result<tonic::Response<super::EnrollUserResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/EnrollUser",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_user_progress(
            &mut self,
            request: impl tonic::IntoRequest<super::GetUserProgressRequest>,
        ) -> Result<tonic::Response<super::GetUserProgressResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetUserProgress",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_lesson_state(
            &mut self,
            request: impl tonic::IntoRequest<super::GetLessonStateRequest>,
        ) -> Result<tonic::Response<super::GetLessonStateResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetLessonState",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn answer_cards(
            &mut self,
            request: impl tonic::IntoRequest<super::AnswerCardsRequest>,
        ) -> Result<tonic::Response<super::AnswerCardsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/AnswerCards",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn create_course(
            &mut self,
            request: impl tonic::IntoRequest<super::CreateCourseRequest>,
        ) -> Result<tonic::Response<super::CreateCourseResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/CreateCourse",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn create_lesson(
            &mut self,
            request: impl tonic::IntoRequest<super::CreateLessonRequest>,
        ) -> Result<tonic::Response<super::CreateLessonResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/CreateLesson",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn update_course(
            &mut self,
            request: impl tonic::IntoRequest<super::UpdateCourseRequest>,
        ) -> Result<tonic::Response<super::UpdateCourseResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/UpdateCourse",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn update_lesson(
            &mut self,
            request: impl tonic::IntoRequest<super::UpdateLessonRequest>,
        ) -> Result<tonic::Response<super::UpdateLessonResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/UpdateLesson",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn sync_state(
            &mut self,
            request: impl tonic::IntoRequest<super::SyncStateRequest>,
        ) -> Result<tonic::Response<super::SyncStateResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/SyncState",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_course_deck_ids(
            &mut self,
            request: impl tonic::IntoRequest<super::GetCourseDeckIdsRequest>,
        ) -> Result<tonic::Response<super::GetCourseDeckIdsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetCourseDeckIds",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_lesson_deck_ids(
            &mut self,
            request: impl tonic::IntoRequest<super::GetLessonDeckIdsRequest>,
        ) -> Result<tonic::Response<super::GetLessonDeckIdsResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetLessonDeckIds",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        /// Backoffice endpoints
        pub async fn get_students_progress(
            &mut self,
            request: impl tonic::IntoRequest<super::GetStudentsProgressRequest>,
        ) -> Result<tonic::Response<super::GetStudentsProgressResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetStudentsProgress",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_user_state(
            &mut self,
            request: impl tonic::IntoRequest<super::GetUserStateRequest>,
        ) -> Result<tonic::Response<super::GetUserStateResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetUserState",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn set_user_state(
            &mut self,
            request: impl tonic::IntoRequest<super::SetUserStateRequest>,
        ) -> Result<tonic::Response<super::SetUserStateResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/SetUserState",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
        pub async fn get_all_courses(
            &mut self,
            request: impl tonic::IntoRequest<super::GetAllCoursesRequest>,
        ) -> Result<tonic::Response<super::GetAllCoursesResponse>, tonic::Status> {
            self.inner
                .ready()
                .await
                .map_err(|e| {
                    tonic::Status::new(
                        tonic::Code::Unknown,
                        format!("Service was not ready: {}", e.into()),
                    )
                })?;
            let codec = tonic::codec::ProstCodec::default();
            let path = http::uri::PathAndQuery::from_static(
                "/course.v1.CourseAPI/GetAllCourses",
            );
            self.inner.unary(request.into_request(), path, codec).await
        }
    }
}
