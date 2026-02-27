import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://julien-sobczak.github.io',
	base: 'the-notewriter',
	integrations: [
		react(),
		starlight({
			title: 'The NoteWriter Documentation',
			logo: {
				// src: './src/assets/logo.svg', // When using a single logo
				light: './src/assets/logo-version-light.svg',
				dark: './src/assets/logo-version-dark.svg',
				alt: 'The NoteWriter Logo',
				// Remove the text title as the logo includes it
				replacesTitle: true,
			},
			social: {
				github: 'https://github.com/julien-sobczak/the-notewriter',
			},
			head: [
				{
					tag: 'link',
					attrs: {
						rel: 'preconnect',
						href: 'https://fonts.googleapis.com',
					},
				},
				{
					tag: 'link',
					attrs: {
						rel: 'preconnect',
						href: 'https://fonts.gstatic.com',
						crossorigin: 'anonymous',
					},
				},
				{
					tag: 'link',
					attrs: {
						rel: 'stylesheet',
						href: 'https://fonts.googleapis.com/css2?family=Inspiration&family=Open+Sans:ital,wdth,wght@0,87.5,300..800;1,87.5,300..800&family=Roboto+Mono:ital,wght@0,100..700;1,100..700&display=swap',
					},
				},
			],
			customCss: ['./src/assets/landing.css', './src/assets/documentation.css'],
			sidebar: [
				{
					label: "Overview",
					link: '/overview'
				},
				{
					label: "Why",
					link: '/why'
				},
				// TODO rename in In a Nutshell/In Depth/In Action
				{
					label: "Getting Started",
					link: '/getting-started'
				},
				{
					label: 'User Guide',
					items: [
						// TODO Rename to Overview if there are other Overview pages
						{ label: 'Introduction', link: '/user-guide/introduction' },
						{ label: 'Configuration', link: '/user-guide/configuration' },
						{ label: 'Files', link: '/user-guide/files' },
						{ label: 'Notes', link: '/user-guide/notes' },
						{ label: 'Attributes', link: '/user-guide/attributes' },
						{ label: 'Tags', link: '/user-guide/tags' },
						{ label: 'Note Types', link: '/user-guide/note-types' },
						// TODO file types with their Markdown schemas
						// { label: 'File Types', link: '/user-guide/file-types' },
						{ label: 'Links', link: '/user-guide/links' },
						{ label: 'Linter', link: '/user-guide/linter' },
						{
							label: 'Objects',
							items: [
								{ label: 'Overview', link: '/user-guide/objects' },
								{ label: 'Medias', link: '/user-guide/medias' },
								{ label: 'Flashcards', link: '/user-guide/flashcards' },
								{ label: 'Gotos', link: '/user-guide/gotos' },
								{ label: 'Reminders', link: '/user-guide/reminders' },
								{ label: 'Memories', link: '/user-guide/memories' },
							]
						},
						{ label: 'Hooks', link: '/user-guide/hooks' },
						{ label: 'Remotes', link: '/user-guide/remotes' },
					],
				},
				{
					label: 'Use Cases',
					items: [
						{ label: 'Overview', link: '/use-cases/overview' },
						{ label: 'Reading', link: '/use-cases/reading' },
						{ label: 'Learning', link: '/use-cases/learning' },
						{ label: 'Journaling', link: '/use-cases/journaling' },
						{ label: 'Planning', link: '/use-cases/planning' },
						{ label: 'Writing', link: '/use-cases/writing' },
						{ label: 'Creating', link: '/use-cases/creating' },
					],
				},
				{
					label: 'Examples',
					items: [
						{ label: "Overview", link: '/examples/my-workflow/overview' },
						{ label: 'My Daily Workflow', link: '/examples/my-workflow/daily' },
						{ label: 'My Reading Workflow', link: '/examples/my-workflow/reading' },
						{ label: 'My Writing Workflow', link: '/examples/my-workflow/writing' },
						// IMPROVEMENT add more example workflows
						// { label: 'My Journaling Workflow', link: '/examples/my-workflow/writing' },
					],
				},
				{
					label: 'Best Practices',
					items: [
						{ label: 'Guidelines', link: '/practices/guidelines' },
						{ label: 'VS Code', link: '/practices/vs-code' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Syntax', link: '/reference/syntax' },
						{ label: 'Configuration', link: '/reference/configuration' },
						{ label: 'Internals', link: '/reference/internals' },
						{
							label: "Commands",
							items: [
								{ label: "nt init", link: '/reference/commands/nt-init' },
								{ label: "nt add", link: '/reference/commands/nt-add' },
								{ label: "nt status", link: '/reference/commands/nt-status' },
								{ label: "nt diff", link: '/reference/commands/nt-diff' },
								{ label: "nt reset", link: '/reference/commands/nt-reset' },
								{ label: "nt commit", link: '/reference/commands/nt-commit' },
								{ label: "nt push", link: '/reference/commands/nt-push' },
								{ label: "nt pull", link: '/reference/commands/nt-pull' },
								{ label: "nt gc", link: '/reference/commands/nt-gc' },
								{ label: "nt lint", link: '/reference/commands/nt-lint' },
								{ label: "nt cat-file", link: '/reference/commands/nt-cat-file' },
								{
									label: "Extras",
									items: [
										{ label: "nt-vault", link: '/reference/commands/nt-vault' },
										{ label: "nt-anki", link: '/reference/commands/nt-anki' },
										{ label: "nt-book", link: '/reference/commands/nt-book' },
									],
								},
							],
						}
					]
				},
				{
					label: 'Developer Guide',
					items: [
						{ label: 'Presentation', link: '/developers/presentation' },
						{ label: 'Principles', link: '/developers/principles' },
						{ label: 'From Scratch', link: '/developers/from-scratch' },
						{ label: 'Guidelines', link: '/developers/guidelines' },
						{ label: 'Contributing', link: '/developers/contributing' },
					]
				}
			],
		}),
	],
});
