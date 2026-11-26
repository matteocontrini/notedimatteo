// Small enhancements for the Go-rendered pages. The initial theme is applied
// by an inline script in the page head, before the first paint.

type Mode = 'system' | 'light' | 'dark';

const modeStorageKey = 'mode-watcher-mode';
const prefersLight = window.matchMedia('(prefers-color-scheme: light)');

function readMode(): Mode {
	try {
		const mode = localStorage.getItem(modeStorageKey);
		if (mode === 'system' || mode === 'light' || mode === 'dark') return mode;
	} catch {
		// Storage can be unavailable, for example with blocked site data
	}
	return 'system';
}

function saveMode(mode: Mode) {
	try {
		localStorage.setItem(modeStorageKey, mode);
	} catch {
		// The choice still applies to this page
	}
}

function applyMode(mode: Mode) {
	const light = mode === 'light' || (mode === 'system' && prefersLight.matches);
	const root = document.documentElement;

	// Switch without animating every color transition, like mode-watcher
	const style = document.createElement('style');
	style.textContent = '* { transition: none !important; }';
	document.head.appendChild(style);

	root.classList.toggle('dark', !light);
	root.style.colorScheme = light ? 'light' : 'dark';

	window.getComputedStyle(document.body).getPropertyValue('opacity');
	style.remove();
}

function setUpThemeSelect() {
	const select = document.querySelector<HTMLSelectElement>('[data-theme-select]');
	if (!select) return;

	select.value = readMode();
	select.addEventListener('change', () => {
		const mode = select.value as Mode;
		saveMode(mode);
		applyMode(mode);
	});

	prefersLight.addEventListener('change', () => {
		if (readMode() === 'system') applyMode('system');
	});
}

function setUpTagSort() {
	const button = document.querySelector<HTMLButtonElement>('[data-tag-sort]');
	const list = document.querySelector<HTMLUListElement>('[data-tag-list]');
	if (!button || !list) return;

	const byCount = Array.from(list.children) as HTMLElement[];
	const byName = [...byCount].sort((a, b) =>
		(a.dataset.name ?? '').localeCompare(b.dataset.name ?? ''),
	);

	button.addEventListener('click', () => {
		const alphabetical = button.getAttribute('aria-pressed') !== 'true';
		button.setAttribute('aria-pressed', String(alphabetical));
		list.replaceChildren(...(alphabetical ? byName : byCount));
	});
}

// Ask for confirmation before submitting a form with data-confirm, like a draft's delete form
function setUpConfirmations() {
	for (const form of document.querySelectorAll<HTMLFormElement>('form[data-confirm]')) {
		form.addEventListener('submit', (event) => {
			if (!confirm(form.dataset.confirm ?? '')) event.preventDefault();
		});
	}
}

// Shorten quotes in listings that are much taller than the collapsed height,
// and add a button that shows the whole quote
function setUpCollapsedQuotes() {
	for (const quote of document.querySelectorAll<HTMLElement>('[data-collapse] > blockquote')) {
		const fullHeight = quote.scrollHeight;
		quote.classList.add('collapsed');
		if (fullHeight < quote.clientHeight * 1.5) {
			quote.classList.remove('collapsed');
			continue;
		}

		const button = document.createElement('button');
		button.type = 'button';
		button.className = 'expand-quote';
		button.textContent = 'Mostra tutto';
		button.addEventListener('click', () => {
			quote.classList.remove('collapsed');
			button.remove();
		});
		quote.after(button);
	}
}

setUpThemeSelect();
setUpTagSort();
setUpConfirmations();
setUpCollapsedQuotes();
