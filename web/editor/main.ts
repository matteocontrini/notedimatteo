// Mounts the post editor on the Go-rendered edit page, which embeds the
// editor's data as JSON (see editPage in internal/server/edit.go).

import { mount } from 'svelte';
import Editor from './Editor.svelte';
import type { EditorData } from './form';

const data = JSON.parse(document.getElementById('editor-data')?.textContent ?? '') as EditorData;
const target = document.getElementById('editor');
if (!target) throw new Error('The page has no #editor element');

mount(Editor, { target, props: { data } });
