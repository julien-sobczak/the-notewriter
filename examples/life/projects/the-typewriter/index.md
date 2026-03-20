---
source: https://github.com/julien-sobczak/the-typewriter
---

# The TypeWriter

## Synopsis: Introduction

_The TypeWriter_ is a desktop application designed to help developers and writers practice and improve their touch typing skills using real-world content. Built with React and Electron, it provides an immersive typing experience using two primary content sources:

1. **Git Repository Mode**: Practice typing by recreating code from popular open-source repositories. This helps developers improve their coding speed while familiarizing themselves with different coding styles and patterns.

2. **ePub Reader Mode**: Practice typing by transcribing books in ePub format. Perfect for improving general typing speed and accuracy while engaging with literature.

The application tracks progress, provides detailed statistics, and adapts difficulty based on performance. It's designed for both casual practice sessions and intensive training.

## List: Useful Links

* [GitHub Repository](https://github.com/julien-sobczak/the-typewriter/ "#go/the-typewriter/github")
* [GitHub Section](https://github.com/julien-sobczak/the-typewriter/${section:[issues,pulls,actions,...]} "#go/the-typewriter/github-section")

## Tasks

### Master: Backlog `#bookmark`

```gotemplate
{{- range query "type:Task" }}
- {{ .ShortTitle }}{{ RenderTags .NoteTags }}{{ RenderAttributes .NoteAttributes }}
{{- end }}
```

### Task: Setup Electron Project Structure 🚨

Initialize the Electron application with React integration. Set up the basic window management, IPC communication, and development environment.

### Task: Implement Git Repository Integration ❗️

Create functionality to clone and parse Git repositories. Extract code files and prepare them for typing exercises.

### Task: Build ePub Parser ❗️

Implement ePub file parsing to extract text content. Handle various ePub formats and preserve basic formatting.

### Task: Design Typing Interface 🔼

Create the main typing interface with syntax highlighting for code, real-time error detection, and smooth scrolling.

### Task: Implement Statistics Tracking 🔼

Build a system to track WPM (words per minute), accuracy, common mistakes, and progress over time. Create data visualization components.

### Task: Add User Settings and Preferences 🔽

Implement settings for theme, font size, difficulty level, and content preferences.

### Task: Create Progress Dashboard 🔼

Design and implement a dashboard showing user progress, achievements, and recommendations.

### Task: Write Unit Tests 🔼

Create comprehensive test coverage for core functionality using Jest and React Testing Library.

### Task: Package Application for Distribution 🔽

Set up electron-builder to create installers for Windows, macOS, and Linux.

## Ideas

### Master: Ideas

```gotemplate
{{- range query "type:Idea" }}
- {{ .ShortTitle }}{{ RenderTags .NoteTags }}{{ RenderAttributes .NoteAttributes }}
{{- end }}
```

### Idea: Custom Content Import

Allow users to import their own Git repositories or text files for personalized practice sessions.

### Idea: Audio Feedback

Add optional sound effects for keystrokes, errors, and achievements to enhance the typing experience.

### Idea: Keyboard Layout Support

Support multiple keyboard layouts (Dvorak, Colemak, etc.) and provide layout-specific training exercises.
