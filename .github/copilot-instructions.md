# GitHub Copilot Custom Instructions for PPO Kanban Board

Project: PPO Kanban Board (Sistem Manajemen Proyek - Lingkungan Perbankan)
Tech Stack: Go 1.24 (Chi, sqlc, pgx/v5, Argon2id) + React 19 (TypeScript, Tailwind CSS, Vite) + PostgreSQL

## Banking Security & Coding Standards
1. Strict Role-Based Access Control (RBAC):
   - Admin: Full system control, User Management.
   - Project Manager: Project creation, assignment, columns.
   - Team Leader: Task execution, status updates.
   - Team Risk (role key: `risk_management`): Risk assessment and mitigation.
   - Team Audit (role key: `audit`): Audit findings and compliance review.
2. Never hardcode credentials, secrets, or tokens.
3. Every write action (create, update, delete) must log to `audit_logs` (Audit Trail) with IP and Request ID.
4. User password policy: minimum 12 characters, Argon2id hashing, `must_change_password` flag enabled on creation.
5. Soft/strict deletion guards: Do not allow admin to delete their own account (`SELF_DELETE_FORBIDDEN`).
6. UI must use Indonesian banking standard terminology with Tailwind CSS clean aesthetic.
