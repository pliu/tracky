document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const authContainer = document.getElementById('auth-container');
    const notebooksContainer = document.getElementById('notebooks-container');
    const notesContainer = document.getElementById('notes-container');
    const loginForm = document.getElementById('login-form');
    const signupForm = document.getElementById('signup-form');
    const tabLogin = document.getElementById('tab-login');
    const tabSignup = document.getElementById('tab-signup');
    const authMessage = document.getElementById('auth-message');
    const logoutBtn = document.getElementById('logout-btn');
    const noteContent = document.getElementById('note-content');
    const createNoteBtn = document.getElementById('create-note-btn');
    const notesList = document.getElementById('notes-list');
    const notebooksList = document.getElementById('notebooks-list');
    const notebookName = document.getElementById('notebook-name');
    const createNotebookBtn = document.getElementById('create-notebook-btn');
    const backToNotebooks = document.getElementById('back-to-notebooks');
    const currentNotebookName = document.getElementById('current-notebook-name');

    // State
    let isLoggedIn = false;
    let currentNotebook = null;
    let expandedDays = new Set();

    // Check initial session
    checkSession();

    // Event Listeners
    tabLogin.addEventListener('click', () => switchTab('login'));
    tabSignup.addEventListener('click', () => switchTab('signup'));

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        await handleAuth('/api/login', { username, password });
    });

    signupForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('signup-username').value;
        const password = document.getElementById('signup-password').value;
        await handleAuth('/api/signup', { username, password }, true);
    });

    logoutBtn.addEventListener('click', async () => {
        await fetch('/api/logout');
        setLoggedIn(false);
    });

    createNoteBtn.addEventListener('click', createNote);
    createNotebookBtn.addEventListener('click', createNotebook);
    backToNotebooks.addEventListener('click', showNotebooks);

    // Functions
    function switchTab(tab) {
        authMessage.textContent = '';
        if (tab === 'login') {
            tabLogin.classList.add('active');
            tabSignup.classList.remove('active');
            loginForm.classList.remove('hidden');
            signupForm.classList.add('hidden');
        } else {
            tabLogin.classList.remove('active');
            tabSignup.classList.add('active');
            loginForm.classList.add('hidden');
            signupForm.classList.remove('hidden');
        }
    }

    async function handleAuth(endpoint, data, isSignup = false) {
        authMessage.textContent = 'Processing...';
        authMessage.className = '';

        try {
            const res = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (res.ok) {
                if (isSignup) {
                    authMessage.textContent = 'Signup successful! Please login.';
                    authMessage.className = 'success';
                    switchTab('login');
                } else {
                    setLoggedIn(true);
                }
            } else {
                const text = await res.text();
                authMessage.textContent = text || 'Error occurred';
                authMessage.className = 'error';
            }
        } catch (err) {
            authMessage.textContent = 'Network error';
            authMessage.className = 'error';
        }
    }

    function setLoggedIn(status) {
        isLoggedIn = status;
        currentNotebook = null;
        if (status) {
            authContainer.classList.add('hidden');
            logoutBtn.classList.remove('hidden');
            showNotebooks();
        } else {
            authContainer.classList.remove('hidden');
            notebooksContainer.classList.add('hidden');
            notesContainer.classList.add('hidden');
            logoutBtn.classList.add('hidden');
            notesList.innerHTML = '';
            notebooksList.innerHTML = '';
            loginForm.reset();
            signupForm.reset();
            authMessage.textContent = '';
        }
    }

    function showNotebooks() {
        currentNotebook = null;
        notesContainer.classList.add('hidden');
        notebooksContainer.classList.remove('hidden');
        fetchNotebooks();
    }

    function showNotes(notebook) {
        currentNotebook = notebook;
        currentNotebookName.textContent = notebook.name;
        notebooksContainer.classList.add('hidden');
        notesContainer.classList.remove('hidden');
        fetchNotes();
    }

    async function checkSession() {
        try {
            const res = await fetch('/api/notebooks');
            if (res.ok) {
                setLoggedIn(true);
            } else {
                setLoggedIn(false);
            }
        } catch (e) {
            setLoggedIn(false);
        }
    }

    async function fetchNotebooks() {
        try {
            const res = await fetch('/api/notebooks');
            if (res.ok) {
                const notebooks = await res.json();
                renderNotebooks(notebooks);
            }
        } catch (e) {
            console.error('Failed to fetch notebooks');
        }
    }

    function renderNotebooks(notebooks) {
        notebooksList.innerHTML = '';
        if (!notebooks || notebooks.length === 0) {
            notebooksList.innerHTML = '<p style="text-align:center; color:var(--text-secondary)">No notebooks yet.</p>';
            return;
        }

        notebooks.forEach(nb => {
            const div = document.createElement('div');
            div.className = 'notebook-card';
            div.innerHTML = `
                <span class="notebook-name">${escapeHtml(nb.name)}</span>
                <button class="delete-notebook-btn" title="Delete">🗑️</button>
            `;
            div.addEventListener('click', () => showNotes(nb));
            div.querySelector('.delete-notebook-btn').addEventListener('click', (e) => {
                e.stopPropagation();
                deleteNotebook(nb.id);
            });
            notebooksList.appendChild(div);
        });
    }

    async function createNotebook() {
        const name = notebookName.value.trim();
        if (!name) return;

        try {
            const res = await fetch('/api/notebooks', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });

            if (res.ok) {
                notebookName.value = '';
                fetchNotebooks();
            }
        } catch (e) {
            console.error('Failed to create notebook');
        }
    }

    async function deleteNotebook(id) {
        if (!confirm('Delete this notebook and all its notes?')) return;
        try {
            const res = await fetch(`/api/notebooks?id=${id}`, { method: 'DELETE' });
            if (res.ok) {
                fetchNotebooks();
            }
        } catch (e) {
            console.error('Failed to delete notebook');
        }
    }

    async function fetchNotes() {
        if (!currentNotebook) return;
        try {
            const res = await fetch(`/api/notes?notebook_id=${currentNotebook.id}`);
            if (res.ok) {
                const notes = await res.json();
                renderNotes(notes);
            }
        } catch (e) {
            console.error('Failed to fetch notes');
        }
    }

    async function createNote() {
        if (!currentNotebook) return;
        const content = noteContent.value.trim();
        if (!content) return;

        try {
            const res = await fetch(`/api/notes?notebook_id=${currentNotebook.id}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ content })
            });

            if (res.ok) {
                noteContent.value = '';
                fetchNotes();
            }
        } catch (e) {
            console.error('Failed to create note');
        }
    }

    async function deleteNote(id) {
        if (!confirm('Delete this note?')) return;
        try {
            const res = await fetch(`/api/notes?id=${id}`, { method: 'DELETE' });
            if (res.ok) {
                fetchNotes();
            }
        } catch (e) {
            console.error('Failed to delete note');
        }
    }

    async function saveEdit(id, textarea, noteCard) {
        const newContent = textarea.value.trim();
        if (newContent === '') return;
        try {
            const res = await fetch(`/api/notes?id=${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ content: newContent })
            });
            if (res.ok) {
                fetchNotes();
            }
        } catch (e) {
            console.error('Failed to edit note');
        }
    }

    function startEdit(noteCard, note) {
        const contentDiv = noteCard.querySelector('.note-content');
        const actionsDiv = noteCard.querySelector('.note-actions');

        contentDiv.classList.add('hidden');
        actionsDiv.classList.add('hidden');

        const editContainer = document.createElement('div');
        editContainer.className = 'edit-container';
        editContainer.innerHTML = `
            <textarea class="edit-textarea">${note.content}</textarea>
            <div class="edit-buttons">
                <button class="save-btn primary-btn">Save</button>
                <button class="cancel-btn">Cancel</button>
            </div>
        `;
        noteCard.appendChild(editContainer);

        const textarea = editContainer.querySelector('.edit-textarea');
        textarea.focus();
        textarea.setSelectionRange(textarea.value.length, textarea.value.length);

        editContainer.querySelector('.save-btn').addEventListener('click', () => saveEdit(note.id, textarea, noteCard));
        editContainer.querySelector('.cancel-btn').addEventListener('click', () => {
            editContainer.remove();
            contentDiv.classList.remove('hidden');
            actionsDiv.classList.remove('hidden');
        });
    }

    function renderNotes(notes) {
        notesList.innerHTML = '';
        if (!notes || notes.length === 0) {
            notesList.innerHTML = '<p style="text-align:center; color:var(--text-secondary)">No notes yet.</p>';
            return;
        }

        const now = new Date();
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const weekAgo = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000);

        const groups = {};
        const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

        notes.forEach(note => {
            const noteDate = new Date(note.created_at);
            const noteDay = new Date(noteDate.getFullYear(), noteDate.getMonth(), noteDate.getDate());
            const dayKey = noteDay.toISOString().split('T')[0];

            if (!groups[dayKey]) {
                groups[dayKey] = { date: noteDay, notes: [] };
            }
            groups[dayKey].notes.push(note);
        });

        const sortedDays = Object.keys(groups).sort((a, b) => new Date(b) - new Date(a));

        sortedDays.forEach(dayKey => {
            const group = groups[dayKey];
            const isToday = group.date.getTime() === today.getTime();
            const isThisWeek = group.date >= weekAgo;

            if (isToday) {
                const header = document.createElement('h3');
                header.className = 'day-header';
                header.textContent = 'Today';
                notesList.appendChild(header);

                group.notes.forEach(note => {
                    notesList.appendChild(createNoteCard(note));
                });
            } else if (isThisWeek) {
                const details = document.createElement('details');
                details.className = 'day-group';
                details.setAttribute('data-day', dayKey);

                if (expandedDays.has(dayKey)) {
                    details.open = true;
                }

                details.addEventListener('toggle', () => {
                    if (details.open) {
                        expandedDays.add(dayKey);
                    } else {
                        expandedDays.delete(dayKey);
                    }
                });

                const summary = document.createElement('summary');
                summary.textContent = dayNames[group.date.getDay()] + ' (' + group.notes.length + ' notes)';
                details.appendChild(summary);

                group.notes.forEach(note => {
                    details.appendChild(createNoteCard(note));
                });

                notesList.appendChild(details);
            } else {
                const details = document.createElement('details');
                details.className = 'day-group';
                details.setAttribute('data-day', dayKey);

                if (expandedDays.has(dayKey)) {
                    details.open = true;
                }

                details.addEventListener('toggle', () => {
                    if (details.open) {
                        expandedDays.add(dayKey);
                    } else {
                        expandedDays.delete(dayKey);
                    }
                });

                const summary = document.createElement('summary');
                summary.textContent = group.date.toLocaleDateString() + ' (' + group.notes.length + ' notes)';
                details.appendChild(summary);

                group.notes.forEach(note => {
                    details.appendChild(createNoteCard(note));
                });

                notesList.appendChild(details);
            }
        });
    }

    function createNoteCard(note) {
        const time = new Date(note.created_at).toLocaleTimeString();
        const div = document.createElement('div');
        div.className = 'note-card';
        div.innerHTML = `
            <div class="note-header">
                <span class="note-meta">${time}</span>
                <div class="note-actions">
                    <button class="edit-btn" title="Edit">✏️</button>
                    <button class="delete-btn" title="Delete">🗑️</button>
                </div>
            </div>
            <div class="note-content">${escapeHtml(note.content)}</div>
        `;
        div.querySelector('.edit-btn').addEventListener('click', () => startEdit(div, note));
        div.querySelector('.delete-btn').addEventListener('click', () => deleteNote(note.id));
        return div;
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
});
