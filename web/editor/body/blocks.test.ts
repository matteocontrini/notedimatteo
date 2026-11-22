import { EditorState } from '@codemirror/state';
import { describe, expect, it } from 'vitest';
import { blockDecorations } from './blocks';
import { markdownLanguage } from './language';

describe('blocks', () => {
	it('mark the lines of quotes and code blocks, and the end of each quote', () => {
		const doc = [
			'text',
			'> one',
			'> two',
			'',
			'```',
			'code',
			'```',
			'',
			'> quote',
			'> > nested',
		].join('\n');
		const state = EditorState.create({ doc, extensions: markdownLanguage });
		const lines: string[] = [];
		const closes: number[] = [];
		blockDecorations(state).between(0, doc.length, (from, _to, decoration) => {
			if (decoration.spec.widget) closes.push(state.doc.lineAt(from).number);
			else lines.push(`${state.doc.lineAt(from).number}: ${decoration.spec.class}`);
		});
		expect(lines).toEqual([
			'2: cm-md-quote cm-md-quote-first',
			'3: cm-md-quote cm-md-quote-last',
			'5: cm-md-code cm-md-code-first',
			'6: cm-md-code',
			'7: cm-md-code cm-md-code-last',
			'9: cm-md-quote cm-md-quote-first',
			'10: cm-md-quote cm-md-quote-last',
		]);
		expect(closes).toEqual([3, 10]);
	});
});
