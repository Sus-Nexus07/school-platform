package routes

import (
	"net/http"
	"school_platform/handlers"
	"school_platform/middleware"
)

func RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/register", handlers.Register)
	mux.HandleFunc("/api/login", handlers.Login)
	mux.HandleFunc("/api/me", middleware.AuthMiddleware(handlers.Me))
	mux.HandleFunc("/api/courses", middleware.AuthMiddleware(handlers.GetCourses))
	mux.HandleFunc("/api/courses/create", middleware.AuthMiddleware(handlers.CreateCourse))
	mux.HandleFunc("/api/courses/", middleware.AuthMiddleware(handlers.CourseRouter))
	mux.HandleFunc("/api/assignments/", middleware.AuthMiddleware(handlers.AssignmentRouter))
	mux.HandleFunc("/api/submissions/", middleware.AuthMiddleware(handlers.SubmissionRouter))
	mux.HandleFunc("/api/admin/users", middleware.AuthMiddleware(handlers.GetAllUsers))
	mux.HandleFunc("/api/admin/stats", middleware.AuthMiddleware(handlers.GetAdminStats))
	mux.HandleFunc("/api/admin/users/", middleware.AuthMiddleware(handlers.AdminUserRouter))

	return mux
}
