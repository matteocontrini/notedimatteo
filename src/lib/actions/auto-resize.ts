const TEXTAREA_BUFFER_PX = 12;

export function autoResize(node: HTMLTextAreaElement, value?: string) {
	const resize = () => {
		node.style.height = 'auto';
		node.style.height = `${node.scrollHeight + TEXTAREA_BUFFER_PX}px`;
	};

	resize();

	node.addEventListener('input', resize);

	return {
		destroy() {
			node.removeEventListener('input', resize);
		},
		update(newValue?: string) {
			// trigger resize when value changes programmatically
			if (newValue !== value) {
				value = newValue;
				resize();
			}
		}
	};
}
