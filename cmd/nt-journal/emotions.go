package main

type Emotion struct {
	Key string
	// Optional Unicode emoji representing the emotion
	Emoji string
	// English description of the emotions
	Title string
	// List of tags on prompts matching this emotion
	Tags []string
}

var Emotions = []*Emotion{
	{
		Key:   "indifferent",
		Emoji: "😐",
		Title: "Indifferent",
		Tags:  []string{}, // match all prompts
	},
	{
		Key:   "Happy",
		Emoji: "😀",
		Title: "Happy",
		Tags:  []string{"doing", "learning"},
	},
	{
		Key:   "Confident",
		Emoji: "💪",
		Title: "Confident",
		Tags:  []string{"planning", "doing"},
	},
	{
		Key:   "Demotivated",
		Emoji: "😣",
		Title: "Demotivated",
		Tags:  []string{"self-esteem", "self-motivating"},
	},
	{
		Key:   "Disappointed",
		Emoji: "😔",
		Title: "Disappointed",
		Tags:  []string{"self-esteem", "being"},
	},
	{
		Key:   "Pessismitic",
		Emoji: "👎",
		Title: "Pessismitic",
		Tags:  []string{"self-discovery", "being", "focusing"},
	},
	{
		Key:   "Optimistic",
		Emoji: "👍",
		Title: "Optimistic",
		Tags:  []string{"problem-solving", "brainstorming", "learning"},
	},
	{
		Key:   "Puzzled",
		Emoji: "🤯",
		Title: "Puzzled",
		Tags:  []string{"problem-solving", "brainstorming"},
	},
	{
		Key:   "Grateful",
		Emoji: "😘",
		Title: "Grateful",
		Tags:  []string{"self-reflection", "being"},
	},
	{
		Key:   "Curious",
		Emoji: "👨‍🎓",
		Title: "Curious",
		Tags:  []string{"learning", "living", "self-reflection"},
	},
	{
		Key:   "Calm",
		Emoji: "🧘",
		Title: "Calm",
		Tags:  []string{"being", "living"},
	},
	{
		Key:   "Excited",
		Emoji: "⚡️",
		Title: "Excited",
		Tags:  []string{"planning", "doing"},
	},
	{
		Key:   "Tired",
		Emoji: "😴",
		Title: "Tired",
		Tags:  []string{"self-reflection", "self-discovery"},
	},
	{
		Key:   "Energetic",
		Emoji: "🏃‍➡️",
		Title: "Energetic",
		Tags:  []string{"doing", "self-improvement"},
	},
	{
		Key:   "Bored",
		Emoji: "🥱",
		Title: "Bored",
		Tags:  []string{"self-discovery", "self-reflection"},
	},
	{
		Key:   "Annoyed",
		Emoji: "😤",
		Title: "Annoyed",
		Tags:  []string{"being", "living"},
	},
	{
		Key:   "Stressed",
		Emoji: "😩",
		Title: "Stressed",
		Tags:  []string{"self-care", "self-esteem"},
	},
	{
		Key:   "Anxious",
		Emoji: "😬",
		Title: "Anxious",
		Tags:  []string{"self-care", "self-esteem", "self-reflection"},
	},
}
