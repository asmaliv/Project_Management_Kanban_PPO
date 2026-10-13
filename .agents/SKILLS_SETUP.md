# Agent Skills Setup Guide

## Installed Skills

### ✅ Ponytail
**Status:** Installed  
**Location:** `.agents/skills/ponytail.md`  
**Purpose:** Makes AI think like a lazy senior dev - the smallest change that fully solves the task  

**Usage:**
- Automatically active in all sessions
- Levels: lite, full (default), ultra
- Use when: refactoring, simplifying, avoiding over-engineering

**Benefits:**
- -53% code, -41% time, -26% cost, -45% tokens (benchmarked)
- Forces simplest solution that actually works
- Never cuts validation, security, or accessibility

---

## Skills Pending Installation

### ✅ Graphify
**Status:** INSTALLED & ACTIVE  
**Repository:** https://github.com/Graphify-Labs/graphify  
**Purpose:** Turn codebase into queryable knowledge graph - understand architecture instantly

**Installation Status:**
- ✅ Python 3.12.10 installed
- ✅ Graphify 0.9.88 installed
- ✅ Skill registered for Kiro
- ✅ Graph built (390 nodes, 986 edges, 14 communities)

**What You Get:**
- Interactive HTML knowledge graph
- Query codebase without reading all files
- Understand how components connect
- 80% fewer tokens per query vs reading raw files

**Usage After Install:**
```
/graphify .                                  # build graph for current folder
/graphify query "how does auth work?"        # query the graph
/graphify path "UserService" "Database"      # find connection between concepts
/graphify explain "RateLimiter"              # explain a concept
```

---

### 🎨 UI-UX Pro Max
**Status:** Requires manual installation  
**Repository:** https://github.com/nextlevelbuilder/ui-ux-pro-max-skill  
**Purpose:** Design intelligence for professional UI/UX - consistent, accessible, beautiful interfaces

**Installation Steps:**
```powershell
# 1. Clone repository to skills folder
cd .agents/skills
git clone https://github.com/nextlevelbuilder/ui-ux-pro-max-skill ui-ux-pro-max

# 2. The skill will be auto-discovered
```

**What You Get:**
- Design system intelligence (colors, typography, spacing)
- Accessibility best practices (WCAG AA compliance)
- Component consistency enforcement
- Responsive design patterns
- Dark mode support

**Usage After Install:**
- Automatically active when building UI components
- Enforces design tokens and consistency
- Suggests improvements for accessibility

---

## Why These Skills?

### Ponytail
- **Problem:** AI often over-engineers solutions with unnecessary abstractions
- **Solution:** Forces simplest working code
- **Impact:** Less code to maintain, faster to understand, fewer bugs

### Graphify  
- **Problem:** Understanding a new codebase requires reading many files
- **Solution:** Knowledge graph shows architecture at a glance
- **Impact:** Faster onboarding, better decisions, avoid breaking changes

### UI-UX Pro Max
- **Problem:** Inconsistent UI, accessibility issues, design debt
- **Solution:** Enforces design system and best practices
- **Impact:** Professional UI, accessible to all users, maintainable design

---

## Current Project Context

This is a **Go + PostgreSQL + React** project for a Kanban board with 5 roles.

**Ponytail** will help:
- Keep backend services simple and focused
- Avoid unnecessary middleware or abstractions
- Prefer standard library over new dependencies

**Graphify** (when installed) will help:
- Understand how auth → user → project → task layers connect
- Find where RBAC rules are enforced
- See audit log integration points

**UI-UX Pro Max** (when installed) will help:
- Consistent design across all 5 role dashboards
- Accessible drag-and-drop on Kanban board
- Professional color system for priority/status indicators

---

## Next Steps

1. ✅ **Ponytail is active now** - try being "lazy" in your requests
2. For **Graphify**: Install Python and run commands above
3. For **UI-UX Pro Max**: Clone the repository when ready to build frontend

---

## References

- Ponytail: https://github.com/dietrichgebert/ponytail
- Graphify: https://github.com/Graphify-Labs/graphify
- UI-UX Pro Max: https://github.com/nextlevelbuilder/ui-ux-pro-max-skill
