import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'The NoteWriter Documentation',
			social: {
				github: 'https://github.com/julien-sobczak/the-notewriter',
			},
			sidebar: [
				{
					label: "Introduction",
					link: '/intro'
				},
				{
					label: "Why",
					link: '/why'
				},
				{
					label: "Getting Started",
					link: '/getting-started'
				},
				{
					label: 'Guides',
					items: [
						{ label: 'Notes', link: '/guides/notes' },
						{ label: 'Attributes', link: '/guides/attributes' },
						{ label: 'Medias', link: '/guides/medias' },
						{ label: 'Links', link: '/guides/links' },
						{ label: 'Flashcards', link: '/guides/flashcards' },
						{ label: 'Reminders', link: '/guides/reminders' },
						{ label: 'Hooks', link: '/guides/hooks' },
						{ label: 'Linter', link: '/guides/linter' },
						{ label: 'Remote', link: '/guides/remote' },
					],
				},
				{
					label: 'Practices',
					items: [
						{ label: 'Guidelines', link: '/practices/guidelines' },
						{ label: 'VS Code', link: '/practices/vs-code' },
						{ label: 'My Workflow', link: '/practices/my-workflow' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Internals', link: '/references/internals' },
						{
							label: "Commands",
							items: [
								{ label: "nt init", link: '/references/commands/nt-init' },
								{ label: "nt add", link: '/references/commands/nt-add' },
								{ label: "nt status", link: '/references/commands/nt-status' },
								{ label: "nt diff", link: '/references/commands/nt-diff' },
								{ label: "nt reset", link: '/references/commands/nt-reset' },
								{ label: "nt commit", link: '/references/commands/nt-commit' },
								{ label: "nt push", link: '/references/commands/nt-push' },
								{ label: "nt pull", link: '/references/commands/nt-pull' },
								{ label: "nt gc", link: '/references/commands/nt-gc' },
								{ label: "nt lint", link: '/references/commands/nt-lint' },
								{ label: "nt cat-file", link: '/references/commands/nt-cat-file' },
							],
						}
					]
				},
				{
					label: 'Developers',
					items: [
						{ label: 'Presentation', link: '/developers/presentation' },
						{ label: 'Principles', link: '/developers/principles' },
						{ label: 'From Scratch', link: '/developers/from-scratch' },
						{ label: 'Contributing', link: '/developers/contributing' },
					]
				}
			],
		}),
	],
});
