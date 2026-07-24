-- Users table (students, teachers, admins all live here)
CREATE TABLE IF NOT EXISTS users (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(150) UNIQUE NOT NULL,
    password    TEXT NOT NULL,
    role        VARCHAR(20) NOT NULL DEFAULT 'student',
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Courses table (classrooms created by teachers)
CREATE TABLE IF NOT EXISTS courses (
    id          SERIAL PRIMARY KEY,
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    teacher_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Enrollments table (who is in which course)
CREATE TABLE IF NOT EXISTS enrollments (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id   INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    enrolled_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, course_id)
);

-- Lessons table (content inside a course)
CREATE TABLE IF NOT EXISTS lessons (
    id          SERIAL PRIMARY KEY,
    course_id   INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title       VARCHAR(200) NOT NULL,
    content     TEXT,
    position    INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Assignments table (tasks teachers give)
CREATE TABLE IF NOT EXISTS assignments (
    id          SERIAL PRIMARY KEY,
    course_id   INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title       VARCHAR(200) NOT NULL,
    description TEXT,
    due_date    TIMESTAMP,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Submissions table (work students turn in)
CREATE TABLE IF NOT EXISTS submissions (
    id            SERIAL PRIMARY KEY,
    assignment_id INTEGER NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content       TEXT,
    grade         INTEGER,
    submitted_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(assignment_id, user_id)
);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_resets (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(64) UNIQUE NOT NULL,
    expires_at  TIMESTAMP NOT NULL,
    used        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Departments (e.g. Science, Arts, Commercial)
CREATE TABLE IF NOT EXISTS departments (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Classes (e.g. SS2 Science A), each belongs to a department
CREATE TABLE IF NOT EXISTS classes (
    id             SERIAL PRIMARY KEY,
    name           VARCHAR(100) NOT NULL,
    department_id  INTEGER NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    created_at     TIMESTAMP DEFAULT NOW()
);

-- Add class_id to users (nullable — only students need this)
ALTER TABLE users ADD COLUMN IF NOT EXISTS class_id INTEGER REFERENCES classes(id) ON DELETE SET NULL;