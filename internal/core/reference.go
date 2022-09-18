package core

type ReferenceType string

const (
	// See https://github.com/zotero/zotero-schema/blob/master/schema.json for inspiration
	TypeArtwork          ReferenceType = "artwork"
	TypeAudioRecording   ReferenceType = "audioRecording"
	TypeBlogPost         ReferenceType = "blogPost"
	TypeBook             ReferenceType = "book"
	TypeBookSection      ReferenceType = "bookSection"
	TypeConferencePaper  ReferenceType = "conferencePaper"
	TypeDocument         ReferenceType = "document"
	TypeFilm             ReferenceType = "film"
	TypeJournalArticle   ReferenceType = "journalArticle"
	TypeMagazineArticle  ReferenceType = "magazineArticle"
	TypeLetter           ReferenceType = "letter"
	TypeNewspaperArticle ReferenceType = "newspaperArticle"
	TypePodcast          ReferenceType = "podcast"
	TypeThesis           ReferenceType = "thesis"
	TypeWebpage          ReferenceType = "webpage"
)

// List of supported types for references
var ReferenceTypes = []ReferenceType{
	TypeArtwork,
	TypeAudioRecording,
	TypeBlogPost,
	TypeBook,
	TypeBookSection,
	TypeConferencePaper,
	TypeDocument,
	TypeFilm,
	TypeJournalArticle,
	TypeMagazineArticle,
	TypeLetter,
	TypeNewspaperArticle,
	TypePodcast,
	TypeThesis,
	TypeWebpage,
}

// Reference represents a single reference.
type Reference interface {
	Type() ReferenceType
	PublicationYear() string
	Attributes() map[string]interface{} // may differ according the reference type.
	AttributesOrder() []string
}

// ReferenceManager retrieves the metadata for references.
type ReferenceManager interface {

	// Search returns the best matching reference.
	Search(query string) (Reference, error)
}
