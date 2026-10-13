# Graphify Installation Guide for Windows

## Prerequisites Check ❌

Python is **not installed** on this system. Follow the steps below to install Python and then Graphify.

---

## Step 1: Install Python 3.10+

### Option A: Using Microsoft Store (Recommended for Windows)
1. Open Microsoft Store
2. Search for "Python 3.12" or "Python 3.11"
3. Click "Get" or "Install"
4. Wait for installation to complete

### Option B: Download from Python.org
1. Go to https://www.python.org/downloads/
2. Download Python 3.12 (latest stable)
3. **IMPORTANT:** During installation, check ✅ "Add Python to PATH"
4. Click "Install Now"

### Verify Installation
Open a **new** PowerShell window and run:
```powershell
python --version
# Should show: Python 3.12.x or similar

pip --version
# Should show: pip 24.x.x or similar
```

---

## Step 2: Install Graphify

Once Python is installed, run these commands in PowerShell:

```powershell
# Install graphify package
pip install --user graphifyy

# Verify installation
python -c "import graphify; print('Graphify installed successfully!')"
```

### If `graphify` command not found after install:
Add Python Scripts folder to PATH:
```powershell
# Check where Scripts folder is
python -c "import sys; print(sys.exec_prefix + '\\Scripts')"

# Add to PATH (one-time, for current session):
$env:Path += ";C:\Users\sdo\AppData\Local\Programs\Python\Python312\Scripts"

# Or permanently via System Settings:
# Settings > System > About > Advanced system settings > Environment Variables
# Edit PATH and add: C:\Users\sdo\AppData\Local\Programs\Python\Python312\Scripts
```

---

## Step 3: Register Graphify Skill

Navigate to project root and run:
```powershell
cd "C:\Users\sdo\Documents\Antigravity\Project Management PPO\Project_Management_Kanban_PPO\Project_Management_Kanban_PPO"

# Register skill for Kiro
graphify install --platform kiro
```

This will create:
- `.agents/skills/graphify/SKILL.md` (or similar structure)

---

## Step 4: Build Your First Graph

Still in project root:
```powershell
# Build graph for the entire project
graphify .

# This will create graphify-out/ folder with:
# - graph.html (interactive visualization)
# - graph.json (queryable graph data)
# - GRAPH_REPORT.md (summary and insights)
```

Expected output:
```
Corpus: X files · ~Y words
  code:     N files (.go .ts .js ...)
  docs:     N files (.md .txt ...)
Extraction: X nodes, Y edges
Graph: X nodes, Y edges, Z communities
```

---

## Step 5: Query the Graph

Once built, you can query:
```powershell
# Ask questions about the codebase
graphify query "how does authentication work?"

# Find path between concepts
graphify path "AuthService" "Database"

# Explain a specific component
graphify explain "SessionManager"
```

Or in Kiro/Antigravity IDE:
```
/graphify .
/graphify query "what connects auth to audit logs?"
```

---

## What to Expect

### Initial Build Time
- **Small project** (<50 files): ~30 seconds
- **Medium project** (50-200 files): 1-3 minutes
- **Large project** (200+ files): 3-10 minutes

Code files are parsed locally with tree-sitter (no LLM, instant).
Docs/PDFs use LLM for semantic extraction (requires API key or uses IDE's model).

### Outputs

**graph.html** - Interactive visualization:
- Click nodes to explore
- Color-coded by community
- Search and filter
- No server needed, open in any browser

**graph.json** - Structured data:
- Nodes (concepts, functions, classes)
- Edges (calls, imports, references)
- Communities (detected subsystems)
- Query-ready format

**GRAPH_REPORT.md** - Human-readable summary:
- "God nodes" (most connected)
- Community labels
- Surprising connections
- Suggested questions

---

## Troubleshooting

### Issue: `graphify: command not found`
**Solution:** Add Scripts folder to PATH (see Step 2)

### Issue: `No module named 'graphify'`
**Solution:** Run `pip install --user graphifyy` again

### Issue: Build takes forever
**Solution:** 
```powershell
# Build code-only (skips docs, faster)
graphify . --code-only

# Or build specific folder
graphify ./backend
```

### Issue: "No supported files found"
**Check:** Are you in the right directory?
```powershell
# Should be in project root
cd "C:\Users\sdo\Documents\Antigravity\Project Management PPO\Project_Management_Kanban_PPO\Project_Management_Kanban_PPO"
graphify .
```

---

## For This Project (PPO Kanban Board)

### What Graphify Will Show You:

**Backend (Go):**
- How `auth → user → project → task` layers connect
- Where RBAC rules are enforced
- Audit log integration points
- Service dependencies

**Recommended First Queries:**
```powershell
graphify query "how does authentication flow work?"
graphify query "where is RBAC enforced?"
graphify query "how are audit logs written?"
graphify path "AuthService" "AuditLog"
graphify path "UserService" "Database"
```

### Benefits for Phase 4+ Development:
- Understand existing structure before adding projects/tasks
- See what touches the database
- Find all RBAC check points
- Avoid breaking existing integrations

---

## Current Status

- [x] Ponytail skill installed
- [ ] **Python not installed yet** ← YOU ARE HERE
- [ ] Graphify package installation pending
- [ ] Graph build pending
- [ ] UI-UX Pro Max pending

---

## Quick Start Summary

Once Python is installed:
```powershell
pip install --user graphifyy
cd "C:\Users\sdo\Documents\Antigravity\Project Management PPO\Project_Management_Kanban_PPO\Project_Management_Kanban_PPO"
graphify install --platform kiro
graphify .
```

Then query:
```powershell
graphify query "show me the architecture"
```

Or just open `graphify-out/graph.html` in your browser! 🚀

---

## References

- Graphify GitHub: https://github.com/Graphify-Labs/graphify
- Documentation: https://docs.graphify.com
- Python Download: https://www.python.org/downloads/
