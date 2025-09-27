import { tick } from 'svelte';

// Vibe coded, no idea what it does

const TEXTAREA_BUFFER_PX = 12;

export function autoResize(node: HTMLTextAreaElement, value?: string) {
	let frame = 0;

	const resize = () => {
		node.style.height = 'auto';
		node.style.height = `${node.scrollHeight + TEXTAREA_BUFFER_PX}px`;
	};

	const scheduleResize = () => {
		cancelAnimationFrame(frame);
		frame = requestAnimationFrame(resize);
	};

	scheduleResize();
	tick().then(scheduleResize);

	const handleInput = () => {
		scheduleResize();
	};

	node.addEventListener('input', handleInput);

	return {
		destroy() {
			cancelAnimationFrame(frame);
			node.removeEventListener('input', handleInput);
		},
		update(newValue?: string) {
			// trigger resize when value changes programmatically
			if (newValue !== value) {
				value = newValue;
				scheduleResize();
			}
		}
	};
}
