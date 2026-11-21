// Small enhancements for the Go-rendered pages. The initial theme is applied
// by an inline script in the page head, before the first paint.

type Mode = 'system' | 'light' | 'dark';

// mode-watcher's key, so preferences saved by the SvelteKit app still apply
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

setUpThemeSelect();
setUpTagSort();
setUpConfirmations();
