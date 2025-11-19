document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const authContainer = document.getElementById('auth-container');
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

    // State
    let isLoggedIn = false;
    let expandedDays = new Set(); // Track which day groups are expanded

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
        if (status) {
            authContainer.classList.add('hidden');
            notesContainer.classList.remove('hidden');
            logoutBtn.classList.remove('hidden');
            fetchNotes();
        } else {
            authContainer.classList.remove('hidden');
            notesContainer.classList.add('hidden');
            logoutBtn.classList.add('hidden');
            notesList.innerHTML = '';
            loginForm.reset();
            signupForm.reset();
            authMessage.textContent = '';
        }
    }

    async function checkSession() {
        try {
            const res = await fetch('/api/notes');
            if (res.ok) {
                setLoggedIn(true);
            } else {
                setLoggedIn(false);
            }
        } catch (e) {
            setLoggedIn(false);
        }
    }

    async function fetchNotes() {
        try {
            const res = await fetch('/api/notes');
            if (res.ok) {
                const notes = await res.json();
                renderNotes(notes);
            }
        } catch (e) {
            console.error('Failed to fetch notes');
        }
    }

    async function createNote() {
        const content = noteContent.value.trim();
        if (!content) return;

        try {
            const res = await fetch('/api/notes', {
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

        // Hide content, show textarea
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

        // Group notes by day
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

        // Sort days from most recent to oldest
        const sortedDays = Object.keys(groups).sort((a, b) => new Date(b) - new Date(a));

        sortedDays.forEach(dayKey => {
            const group = groups[dayKey];
            const isToday = group.date.getTime() === today.getTime();
            const isThisWeek = group.date >= weekAgo;

            if (isToday) {
                // Today's notes - show directly
                const header = document.createElement('h3');
                header.className = 'day-header';
                header.textContent = 'Today';
                notesList.appendChild(header);

                group.notes.forEach(note => {
                    notesList.appendChild(createNoteCard(note));
                });
            } else if (isThisWeek) {
                // This week's notes - collapsible
                const details = document.createElement('details');
                details.className = 'day-group';
                details.setAttribute('data-day', dayKey);

                // Restore expanded state
                if (expandedDays.has(dayKey)) {
                    details.open = true;
                }

                // Track toggle state
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
                // Older notes - collapsible with date
                const details = document.createElement('details');
                details.className = 'day-group';
                details.setAttribute('data-day', dayKey);

                // Restore expanded state
                if (expandedDays.has(dayKey)) {
                    details.open = true;
                }

                // Track toggle state
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
