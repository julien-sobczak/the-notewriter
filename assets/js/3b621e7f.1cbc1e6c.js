"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[3490],{3905:(e,t,n)=>{n.d(t,{Zo:()=>d,kt:()=>k});var a=n(7294);function i(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function r(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);t&&(a=a.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,a)}return n}function o(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?r(Object(n),!0).forEach((function(t){i(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):r(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function l(e,t){if(null==e)return{};var n,a,i=function(e,t){if(null==e)return{};var n,a,i={},r=Object.keys(e);for(a=0;a<r.length;a++)n=r[a],t.indexOf(n)>=0||(i[n]=e[n]);return i}(e,t);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(a=0;a<r.length;a++)n=r[a],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(i[n]=e[n])}return i}var s=a.createContext({}),p=function(e){var t=a.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):o(o({},t),e)),n},d=function(e){var t=p(e.components);return a.createElement(s.Provider,{value:t},e.children)},m="mdxType",c={inlineCode:"code",wrapper:function(e){var t=e.children;return a.createElement(a.Fragment,{},t)}},u=a.forwardRef((function(e,t){var n=e.components,i=e.mdxType,r=e.originalType,s=e.parentName,d=l(e,["components","mdxType","originalType","parentName"]),m=p(n),u=i,k=m["".concat(s,".").concat(u)]||m[u]||c[u]||r;return n?a.createElement(k,o(o({ref:t},d),{},{components:n})):a.createElement(k,o({ref:t},d))}));function k(e,t){var n=arguments,i=t&&t.mdxType;if("string"==typeof e||i){var r=n.length,o=new Array(r);o[0]=u;var l={};for(var s in t)hasOwnProperty.call(t,s)&&(l[s]=t[s]);l.originalType=e,l[m]="string"==typeof e?e:i,o[1]=l;for(var p=2;p<r;p++)o[p]=n[p];return a.createElement.apply(null,o)}return a.createElement.apply(null,n)}u.displayName="MDXCreateElement"},1142:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>s,contentTitle:()=>o,default:()=>c,frontMatter:()=>r,metadata:()=>l,toc:()=>p});var a=n(7462),i=(n(7294),n(3905));const r={sidebar_position:1},o="Presentation",l={unversionedId:"developers/presentation",id:"developers/presentation",title:"Presentation",description:"The NoteWriter is a CLI to generates notes from files.",source:"@site/docs/developers/presentation.md",sourceDirName:"developers",slug:"/developers/presentation",permalink:"/the-notewriter/docs/developers/presentation",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/developers/presentation.md",tags:[],version:"current",sidebarPosition:1,frontMatter:{sidebar_position:1},sidebar:"documentationSidebar",previous:{title:"Developers",permalink:"/the-notewriter/docs/category/developers"},next:{title:"Principles",permalink:"/the-notewriter/docs/developers/principles"}},s={},p=[{value:"Code Organization",id:"code-organization",level:2},{value:"Implementation",id:"implementation",level:2},{value:"Core (<code>internal/core</code>)",id:"core-internalcore",level:3},{value:"Linter <code>internal/core/lint.go</code>",id:"linter-internalcorelintgo",level:3},{value:"Media (<code>internal/medias</code>)",id:"medias",level:3},{value:"Testing",id:"testing",level:2},{value:"F.A.Q.",id:"faq",level:2},{value:"How to migrate SQL schema",id:"how-to-migrate-sql-schema",level:3},{value:"How to use transactions with SQLite",id:"how-to-use-transactions-with-sqlite",level:3}],d={toc:p},m="wrapper";function c(e){let{components:t,...n}=e;return(0,i.kt)(m,(0,a.Z)({},d,n,{components:t,mdxType:"MDXLayout"}),(0,i.kt)("h1",{id:"presentation"},"Presentation"),(0,i.kt)("p",null,(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," is a CLI to generates notes from files."),(0,i.kt)("p",null,"Users edit a collection of files using a documented syntax (Markdown with a few extensions). ",(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," parses these files to extract objects (note, flashcard, reminder, etc.) from these raw files."),(0,i.kt)("h2",{id:"code-organization"},"Code Organization"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},'.\n\u251c\u2500\u2500 cmd             # Viper commands\n\u251c\u2500\u2500 internal        # The NoteWriter-specific code\n\u2502   \u251c\u2500\u2500 core        # Main logic\n\u2502   \u251c\u2500\u2500 medias      # Media processing\n\u2502   \u251c\u2500\u2500 reference   # Reference processing\n\u2502   \u2514\u2500\u2500 testutil    # Test utilities\n\u2514\u2500\u2500 pkg             # "Reusable" code (not specific to The NoteWriter)\n')),(0,i.kt)("admonition",{type:"tip"},(0,i.kt)("p",{parentName:"admonition"},"Start with commands under ",(0,i.kt)("inlineCode",{parentName:"p"},"cmd/")," when inspecting code to quickly locate the interesting lines of code.")),(0,i.kt)("p",null,"The repository also contains additional directories not directly related to the implementation:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},".\n\u251c\u2500\u2500 build      # Binary built using the Makefile\n\u251c\u2500\u2500 example    # A demo collection of notes\n\u2514\u2500\u2500 website    # The documentation\n")),(0,i.kt)("h2",{id:"implementation"},"Implementation"),(0,i.kt)("h3",{id:"core-internalcore"},"Core (",(0,i.kt)("inlineCode",{parentName:"h3"},"internal/core"),")"),(0,i.kt)("p",null,"Most of the code (and most of the tests) is present in this package."),(0,i.kt)("p",null,"A ",(0,i.kt)("inlineCode",{parentName:"p"},"Collection")," (",(0,i.kt)("inlineCode",{parentName:"p"},"collection.go"),") is the parent container. A ",(0,i.kt)("em",{parentName:"p"},"collection")," traverses directories to find Markdown ",(0,i.kt)("inlineCode",{parentName:"p"},"File")," (",(0,i.kt)("inlineCode",{parentName:"p"},"file.go"),"). A ",(0,i.kt)("em",{parentName:"p"},"file")," can contains ",(0,i.kt)("inlineCode",{parentName:"p"},"Note")," defined using Markdown headings (",(0,i.kt)("inlineCode",{parentName:"p"},"note.go"),"), some of which can be ",(0,i.kt)("inlineCode",{parentName:"p"},"Flashcard")," when using the corresponding kind (",(0,i.kt)("inlineCode",{parentName:"p"},"flashcard.go"),"), ",(0,i.kt)("inlineCode",{parentName:"p"},"Media")," resources referenced using Markdown link (",(0,i.kt)("inlineCode",{parentName:"p"},"media.go"),"), special ",(0,i.kt)("inlineCode",{parentName:"p"},"Link")," when using convention on Markdown link's titles (",(0,i.kt)("inlineCode",{parentName:"p"},"link.go"),"), and ",(0,i.kt)("inlineCode",{parentName:"p"},"Reminder")," when using special tags (",(0,i.kt)("inlineCode",{parentName:"p"},"reminder.go"),")."),(0,i.kt)("p",null,(0,i.kt)("inlineCode",{parentName:"p"},"File"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Note"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Flashcard"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Media"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Link"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Reminder")," represents the ",(0,i.kt)("inlineCode",{parentName:"p"},"Object")," (",(0,i.kt)("inlineCode",{parentName:"p"},"object.go"),") managed by ",(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," and stored inside ",(0,i.kt)("inlineCode",{parentName:"p"},".nt/objects")," indirectly using commits. (Blobs are also stored inside this directory.)"),(0,i.kt)("p",null,"The method ",(0,i.kt)("inlineCode",{parentName:"p"},"walk")," defined on ",(0,i.kt)("inlineCode",{parentName:"p"},"Collection")," makes easy to find files to process (= non-ignorable Markdown files):"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go"},'import (\n    "fmt"\n    "github.com/julien-sobczak/the-notewriter/internal/core"\n)\n\nc := core.CurrentCollection()\nerr := c.walk(paths, func(path string, stat fs.FileInfo) error {\n    relativePath, err := c.GetFileRelativePath(path)\n    if err != nil {\n        return err\n    }\n    fmt.Printf("Found %s", relativePath)\n}\n')),(0,i.kt)("admonition",{type:"info"},(0,i.kt)("p",{parentName:"admonition"},(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," relies heavily on ",(0,i.kt)("a",{parentName:"p",href:"https://en.wikipedia.org/wiki/Singleton_pattern"},"singletons"),". Most of the most abstractions (",(0,i.kt)("inlineCode",{parentName:"p"},"Collection"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"DB"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"Config")," can be retrieved using methods ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentCollection()"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentDB()"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentConfig()")," to easily find a note, persist changes in database, or read configuration settings anywhere in the code. (Singletons are only initialized on first use.)"),(0,i.kt)("p",{parentName:"admonition"},(0,i.kt)("strong",{parentName:"p"},"This strongly differs from most enterprise applications")," where layers and dependency injection are used to have a clean separation of concerns."),(0,i.kt)("p",{parentName:"admonition"},(0,i.kt)("strong",{parentName:"p"},(0,i.kt)("em",{parentName:"strong"},"The NoteWriter")," is a CLI to execute short-lived commands"),' (one execution = one "transaction") where traditional applications process transactions in parallel (one request = one transaction).')),(0,i.kt)("p",null,"All objects must satisfy this interface:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=core/object.go",title:"core/object.go"},"type Object interface {\n    // Kind returns the object kind to determine which kind of object to create.\n    Kind() string\n    // UniqueOID returns the OID of the object.\n    UniqueOID() string\n    // ModificationTime returns the last modification time.\n    ModificationTime() time.Time\n\n    // SubObjects returns the objects directly contained by this object.\n    SubObjects() []StatefulObject\n    // Blobs returns the optional blobs associated with this object.\n    Blobs() []*BlobRef\n    // Relations returns the relations where the current object is the source.\n    Relations() []*Relation\n\n    // Read rereads the object from YAML.\n    Read(r io.Reader) error\n    // Write writes the object to YAML.\n    Write(w io.Writer) error\n}\n")),(0,i.kt)("p",null,"This interface makes easy to factorize common logic betwen objects (ex: all objects can reference other objects and be dumped to YAML inside ",(0,i.kt)("inlineCode",{parentName:"p"},".nt/objects"),")."),(0,i.kt)("p",null,"Each ",(0,i.kt)("em",{parentName:"p"},"object")," is uniquely defined by an OID (a 40-character string) randomly generated from a UUID (see ",(0,i.kt)("inlineCode",{parentName:"p"},"NewOID()"),"), except in tests where the generation is reproducible."),(0,i.kt)("admonition",{type:"tip"},(0,i.kt)("p",{parentName:"admonition"},"Use the ",(0,i.kt)("a",{parentName:"p",href:"/the-notewriter/docs/reference/commands/nt-cat-file"},"command ",(0,i.kt)("inlineCode",{parentName:"a"},"nt cat-file <oid>"))," to find the object from an OID.")),(0,i.kt)("p",null,"Each ",(0,i.kt)("em",{parentName:"p"},"object")," can be ",(0,i.kt)("inlineCode",{parentName:"p"},"Read()")," from a YAML document and ",(0,i.kt)("inlineCode",{parentName:"p"},"Write()")," to a YAML document using the common Go abstractions ",(0,i.kt)("inlineCode",{parentName:"p"},"io.Reader")," and ",(0,i.kt)("inlineCode",{parentName:"p"},"io.Writer"),"."),(0,i.kt)("p",null,"Each ",(0,i.kt)("em",{parentName:"p"},"object")," can contains ",(0,i.kt)("inlineCode",{parentName:"p"},"SubObjects()"),", for example, a ",(0,i.kt)("em",{parentName:"p"},"file")," can contains ",(0,i.kt)("em",{parentName:"p"},"notes"),", or ",(0,i.kt)("inlineCode",{parentName:"p"},"Blobs()"),", which are binary files generated from ",(0,i.kt)("a",{parentName:"p",href:"#medias"},"medias"),", and can references other objects through ",(0,i.kt)("inlineCode",{parentName:"p"},"Relations()"),", for example, a note can use the special attribute ",(0,i.kt)("inlineCode",{parentName:"p"},"@references")," to notify the note is quoted elsewhere. These methods make easy for the ",(0,i.kt)("em",{parentName:"p"},"collection")," to process graphs of objects without having the inspect their types."),(0,i.kt)("p",null,"These ",(0,i.kt)("em",{parentName:"p"},"objects")," must also be stored in a relational database using SQLite. An additional interface must be satisfied for these objects:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/object.go",title:"internal/core/object.go"},'// State describes an object status.\ntype State string\n\nconst (\n    None     State = "none"\n    Added    State = "added"\n    Modified State = "modified"\n    Deleted  State = "deleted"\n)\n\n// StatefulObject to represent the subset of updatable objects persisted in database.\ntype StatefulObject interface {\n    Object\n\n    Refresh() (bool, error)\n\n    // State returns the current state.\n    State() State\n    // ForceState marks the object in the given state\n    ForceState(newState State)\n\n    // Save persists to DB\n    Save() error\n}\n')),(0,i.kt)("p",null,"These ",(0,i.kt)("em",{parentName:"p"},"stateful objects")," must implement the method ",(0,i.kt)("inlineCode",{parentName:"p"},"Save()")," (which will commnly use the singleton ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentDB()")," to retrieve a connection to the database). This method will check the ",(0,i.kt)("inlineCode",{parentName:"p"},"State()")," to determine if the object must be saved using a query ",(0,i.kt)("inlineCode",{parentName:"p"},"INSERT"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"UPDATE"),", or ",(0,i.kt)("inlineCode",{parentName:"p"},"DELETE"),". If no changes have been done, the method ",(0,i.kt)("inlineCode",{parentName:"p"},"Save")," must still update the value of the field ",(0,i.kt)("inlineCode",{parentName:"p"},"LastCheckedAt")," (= useful to detect dead rows in database, that is the rows that represents objects that are no longer present in files)."),(0,i.kt)("p",null,"The method ",(0,i.kt)("inlineCode",{parentName:"p"},"Refresh()")," requires an object to determine if its content is still up-to-date. For example, notes can include other notes using the syntax ",(0,i.kt)("inlineCode",{parentName:"p"},"![[wikilink#note]]"),". When a included note is edited, all notes including it must be refreshed to update their content too."),(0,i.kt)("admonition",{type:"tip"},(0,i.kt)("p",{parentName:"admonition"},"All ",(0,i.kt)("em",{parentName:"p"},"objects")," are parsed from raw Markdown files. To make the parsing logic easily testable, the logic is split in two successive steps:"),(0,i.kt)("pre",{parentName:"admonition"},(0,i.kt)("code",{parentName:"pre"},"Raw Markdown > Parsed Object > (Stateful) Object\n")),(0,i.kt)("p",{parentName:"admonition"},"For example, a ",(0,i.kt)("inlineCode",{parentName:"p"},"File")," can be created from a ",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedFile")," (",(0,i.kt)("inlineCode",{parentName:"p"},"file.go"),") that is created from a raw Markdown document:"),(0,i.kt)("pre",{parentName:"admonition"},(0,i.kt)("code",{parentName:"pre",className:"language-go"},'parsedFile, err := core.ParseFile("notes.md")\n// easy to test the parsing logic with minimal dependencies\n\nfile := NewFileFromParsedFile(nil, parsedFile)\n')),(0,i.kt)("p",{parentName:"admonition"},"The same principle is used for ",(0,i.kt)("em",{parentName:"p"},"notes")," (",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedNote"),") and ",(0,i.kt)("em",{parentName:"p"},"medias")," (",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedMedia"),").")),(0,i.kt)("h3",{id:"linter-internalcorelintgo"},"Linter ",(0,i.kt)("inlineCode",{parentName:"h3"},"internal/core/lint.go")),(0,i.kt)("p",null,"The ",(0,i.kt)("a",{parentName:"p",href:"/the-notewriter/docs/reference/commands/nt-lint"},"command ",(0,i.kt)("inlineCode",{parentName:"a"},"nt lint"))," check for violations. All files are inspected (rules may have changed even if files haven't been modified). The linter reuses the method ",(0,i.kt)("inlineCode",{parentName:"p"},"walk")," to traverse the ",(0,i.kt)("em",{parentName:"p"},"collection"),". The linter doesn't bother with well-formed objects and reuses the type ",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedFile"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedNote"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"ParsedMedia")," to find errors."),(0,i.kt)("p",null,"Each rule is defined using the type ",(0,i.kt)("inlineCode",{parentName:"p"},"LintRule"),":"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/lint.go",title:"internal/core/lint.go"},"type LintRule func(*ParsedFile, []string) ([]*Violation, error)\n")),(0,i.kt)("p",null,"For example, we can write a custom rule (not supported) to validate a file doesn't contains more than 100 notes."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go"},'func MyCustomRule(file *ParsedFile, args []string) ([]*Violation, error) {\n    var violations []*Violation\n\n    notes := ParseNotes(file.Body)\n    if len(notes) > 100 {\n        violations = append(violations, &Violation{\n            Name:         "my-custom-rule",\n            RelativePath: file.RelativePath,\n            Message:      "too many notes",\n        })\n    }\n\n    return violations, nil\n}\n')),(0,i.kt)("p",null,"Each rule must be declared in the global variable ",(0,i.kt)("inlineCode",{parentName:"p"},"LintRules")," in the same file:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/lint.go",title:"internal/core/lint.go"},'var LintRules = map[string]LintRuleDefinition{\n    // ...\n    "my-custon-rule": {\n        Eval: MyCustomRule,\n    },\n}\n')),(0,i.kt)("h3",{id:"medias"},"Media (",(0,i.kt)("inlineCode",{parentName:"h3"},"internal/medias"),")"),(0,i.kt)("p",null,"Medias are static files included in notes using the image syntax:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-md"},"![](audio.wav)\n![](picture.png)\n![](video.mp4)\n")),(0,i.kt)("p",null,"When processing these ",(0,i.kt)("em",{parentName:"p"},"medias"),", ",(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," will create blobs inside the directory ",(0,i.kt)("inlineCode",{parentName:"p"},".nt/objects/"),". The OID is the SHA1 determined from the file content."),(0,i.kt)("p",null,"Images, videos, sounds are processed and are not duplicated. Indeed, ",(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," will optimise these medias like this:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},"Images are converted to AVIF in different sizes (preview = mobile and grid view, large = full-size view, original = original size)."),(0,i.kt)("li",{parentName:"ul"},"Audios are converted to MP3."),(0,i.kt)("li",{parentName:"ul"},"Videos are converted to WebM and a preview image is generated from the first frame.")),(0,i.kt)("p",null,"The AVIF, MP3, and WebM formats are used for their great compression performance and their support (including mobile devices)."),(0,i.kt)("p",null,"By default, ",(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," uses the ",(0,i.kt)("a",{parentName:"p",href:"https://ffmpeg.org/"},"external command ",(0,i.kt)("inlineCode",{parentName:"a"},"ffmpeg"))," (",(0,i.kt)("inlineCode",{parentName:"p"},"internal/medias/ffmpeg"),") to convert and resize medias. All converters must satisfy this interface:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/medias/converters.go",title:"internal/medias/converters.go"},"type Converter interface {\n    ToAVIF(src, dest string, dimensions Dimensions) error\n    ToMP3(src, dest string) error\n    ToWebM(src, dest string) error\n}\n")),(0,i.kt)("p",null,"For example, we can draft a note including a large picture:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell"},'$ mkdir notes\n$ cd notes\n$ echo "# My Notes\\n\\n## Artwork: Whale\\n\\n![](medias/whale.jpg)" > notes.md\n$ nt init\n$ nt add .\n$ nt commit\n[9ef2100625aa4d5c913b8010516fb9a1cd6add98]\n 3 objects changes, 3 insertion(s)\n create file "notes.md" [10a76fcada5a4336bb427b68f23d9690b5ebec33]\n create note "Artwork: Whale" [fed3aa2ace7a4fcb889af7f149bda0d6c802cf43]\n create media medias/whale.jpg [72f94476596d47568e617292ab93e02b64032159]\n$ notes nt cat-file 72f94476596d47568e617292ab93e02b64032159\noid: 72f94476596d47568e617292ab93e02b64032159\nrelative_path: medias/whale.jpg\nkind: picture\ndangling: false\nextension: .jpg\nmtime: 2023-01-01T12:00\nhash: 27198c1682772f01d006b19d4a15018463b7004a\nsize: 6296968\nmode: 420\nblobs:\n    - oid: 5ac8980e0206c51e113191f1cfa4aab3e40b671a\n      mime: image/avif\n      tags:\n        - preview\n        - lossy\n    - oid: 40100b2a68ecf7048566a901d6766be8f85ed186\n      mime: image/avif\n      tags:\n        - large\n        - lossy\n    - oid: 7b4bf88e47e7f782ae9b11e89414d4f66782eeea\n      mime: image/avif\n      tags:\n        - original\n        - lossy\ncreated_at: 2023-01-01T12:00\nupdated_at: 2023-01-01T12:00\n')),(0,i.kt)("p",null,"You can open the generated file. Ex (MacOS):"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-shell"},"$ open -a Preview .nt/objects/5a/5ac8980e0206c51e113191f1cfa4aab3e40b671a\n")),(0,i.kt)("h2",{id:"testing"},"Testing"),(0,i.kt)("p",null,(0,i.kt)("em",{parentName:"p"},"The NoteWriter")," works with files. Testing the application by mocking interactions with the file system would be cumbersome."),(0,i.kt)("admonition",{type:"tip"},(0,i.kt)("p",{parentName:"admonition"},"Almost all tests interacts with the file system and executes SQL queries on a SQLite database instance. Their execution time on a SSD machine are relatively low (10s to run ~500 tests)."),(0,i.kt)("p",{parentName:"admonition"},"Only external commands like ",(0,i.kt)("inlineCode",{parentName:"p"},"ffmpeg")," are impersonated by the test binary file (popular technique used by Golang to test ",(0,i.kt)("inlineCode",{parentName:"p"},"exec")," package).")),(0,i.kt)("p",null,"The package ",(0,i.kt)("inlineCode",{parentName:"p"},"internal/testutil")," exposes various functions to duplicate a directory that are reused by functions inside ",(0,i.kt)("inlineCode",{parentName:"p"},"internal/core/core_test.go")," to provide a complete note collection:"),(0,i.kt)("ol",null,(0,i.kt)("li",{parentName:"ol"},"Copy Markdown files present under ",(0,i.kt)("inlineCode",{parentName:"li"},"internal/core/testdata")," (aka golden files)."),(0,i.kt)("li",{parentName:"ol"},"Init a valid ",(0,i.kt)("inlineCode",{parentName:"li"},".nt")," directory and ensure ",(0,i.kt)("inlineCode",{parentName:"li"},"CurrentCollection()")," reads from this repository."),(0,i.kt)("li",{parentName:"ol"},"Return the temporary directory (automatically cleaned after the test completes)")),(0,i.kt)("p",null,"Example (",(0,i.kt)("inlineCode",{parentName:"p"},"SetUpCollectionFromGoldenDirNamed"),"):"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go"},'package core\n\nimport (\n    "testing"\n\n    "github.com/stretchr/testify/assert"\n    "github.com/stretchr/testify/require"\n)\n\nfunc TestCommandAdd(t *testing.T) {\n    SetUpCollectionFromGoldenDirNamed(t, "TestMinimal")\n\n    err := CurrentCollection().Add("go.md")\n    require.NoError(t, err)\n}\n')),(0,i.kt)("p",null,"Various methods exist:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"SetUpCollectionFromGoldenFile")," initializes a collection containing a single file named after the test (",(0,i.kt)("inlineCode",{parentName:"li"},"TestCommandAdd")," => ",(0,i.kt)("inlineCode",{parentName:"li"},"testdata/TestCommandAdd.md"),")."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"SetUpCollectionFromGoldenFileNamed")," is identical to previous function but accepts the file name."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"SetUpCollectionFromGoldenDir")," initializes a collection from a directory named after the test (",(0,i.kt)("inlineCode",{parentName:"li"},"TestCommandAdd")," => ",(0,i.kt)("inlineCode",{parentName:"li"},"testdata/TestCommandAdd/"),")."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"SetUpCollectionFromGoldenDirNamed")," is identical to previous function but accepts the directory name.")),(0,i.kt)("admonition",{type:"tip"},(0,i.kt)("p",{parentName:"admonition"},"Most tests reuse a common fixture like ",(0,i.kt)("inlineCode",{parentName:"p"},"internal/core/testdata/TestMinimal/")," (= minimal number of files to demonstrate the maximum of features). Indeed, setting up Markdown files for evert test would represent many lines of Markdown fixtures to maintain. The recommendation is to reuse ",(0,i.kt)("inlineCode",{parentName:"p"},"TestMinimal")," as much as possible when the logic is independant but create a custom test fixture when testing special cases."),(0,i.kt)("p",{parentName:"admonition"},"Here are the common fixtures:"),(0,i.kt)("ul",{parentName:"admonition"},(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestMinimal"),": A basic set of files using most of the features."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestMedias"),": A basic set of files using all supported medias file types."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestPostProcessing"),": A basic set exposing all post-processing rules applies to raw notes."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestLint"),": A basic set exposing violations for every rules."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestRelations"),": A basic set of inter-referenced notes.")),(0,i.kt)("p",{parentName:"admonition"},"Here are some specific fixtures: (\u26a0\ufe0f be careful when reusing them)"),(0,i.kt)("ul",{parentName:"admonition"},(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestIgnore"),": A basic set with ignorable files and notes."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestInheritance"),": A basic set with inheritable and non-inheritable attributes between files and notes."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"TestNoteFTS"),": A basic set to demonstrate the full-text search with SQLite."))),(0,i.kt)("p",null,"In addition, several utilities are sometimes required to make tests reproductible:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"FreezeNow()")," and ",(0,i.kt)("inlineCode",{parentName:"li"},"FreezeAt(time.Time)")," ensures successive calls to ",(0,i.kt)("inlineCode",{parentName:"li"},"clock.Now()")," returns a precise timestamp."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"SetNextOIDs(...string)"),", ",(0,i.kt)("inlineCode",{parentName:"li"},"UseFixedOID(string)"),", and ",(0,i.kt)("inlineCode",{parentName:"li"},"UseSequenceOID()")," ensures generated OIDs are deterministic (using respectively a predefined sequence of OIDs, the same OIDs, or OIDs incremented by 1).")),(0,i.kt)("p",null,"All these test helpers restores the initial configuration using ",(0,i.kt)("inlineCode",{parentName:"p"},"t.Cleanup()"),"."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go"},"func TestHelpers(t *testing.T) {\n    SetUpCollectionFromTempDir(t) // empty collection\n\n    UseSequenceOID(t) // 0000000000000000000000000000000000000001\n                      // 0000000000000000000000000000000000000002\n                      // ...\n    FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))\n    // clock.Now() will now always return 2023-01-1T12:30:00Z\n\n    ...\n}\n")),(0,i.kt)("h2",{id:"faq"},"F.A.Q."),(0,i.kt)("h3",{id:"how-to-migrate-sql-schema"},"How to migrate SQL schema"),(0,i.kt)("p",null,"When the method ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentDB().Client()")," is first called, the SQL database is read to initialize the connection. Then, the code uses ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/golang-migrate/migrate/v4"},(0,i.kt)("inlineCode",{parentName:"a"},"golang-migrate"))," to determine if migrations (",(0,i.kt)("inlineCode",{parentName:"p"},"internal/core/sql/*.sql"),") must be run."),(0,i.kt)("h3",{id:"how-to-use-transactions-with-sqlite"},"How to use transactions with SQLite"),(0,i.kt)("p",null,"Use ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentDB().Client()")," to retrieve a valid connection to the SQLite database stored in ",(0,i.kt)("inlineCode",{parentName:"p"},".nt/database.db"),"."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/note.go",title:"internal/core/note.go"},"func (c *Collection) CountNotes() (int, error) {\n    var count int\n    if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note`).Scan(&count); err != nil {\n        return 0, err\n    }\n\n    return count, nil\n}\n")),(0,i.kt)("p",null,"Sometimes, you may want to use transactions. For example, when using ",(0,i.kt)("inlineCode",{parentName:"p"},"nt add"),", if an error occurs when reading a corrupted file, we want to rollback changes to left the database intact. The ",(0,i.kt)("inlineCode",{parentName:"p"},"DB")," exposes methods ",(0,i.kt)("inlineCode",{parentName:"p"},"BeginTransaction()"),", ",(0,i.kt)("inlineCode",{parentName:"p"},"RollbackTransaction()"),", and ",(0,i.kt)("inlineCode",{parentName:"p"},"CommitTransaction()")," for this purpose. Other methods continue to use ",(0,i.kt)("inlineCode",{parentName:"p"},"CurrentDB().Client()")," to create the connection; if a transaction is currently in progress, it will be returned."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/collection.go",title:"internal/core/collection.go"},"func (c *Collection) Add(paths ...string) error {\n    // Run all queries inside the same transaction\n    err = db.BeginTransaction()\n    if err != nil {\n        return err\n    }\n    defer db.RollbackTransaction()\n\n    // Traverse all given path to add files\n    c.walk(paths, func(path string, stat fs.FileInfo) error {\n        // Do changes in database\n    }\n\n    // Don't forget to commit\n    if err := db.CommitTransaction(); err != nil {\n        return err\n    }\n\n    return nil\n}\n")),(0,i.kt)("p",null,"Often, the commands update the relational SQLite database and various files inside ",(0,i.kt)("inlineCode",{parentName:"p"},".nt")," like ",(0,i.kt)("inlineCode",{parentName:"p"},".nt/index"),". The implemented approach is to write files just after committing the SQL transaction to minimize the risk:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-go",metastring:"title=internal/core/collection.go",title:"internal/core/collection.go"},"func (c *Collection) Add(paths ...string) error {\n    ...\n\n    if err := db.CommitTransaction(); err != nil {\n        return err\n    }\n    if err := db.index.Save(); err != nil {\n        return err\n    }\n\n    return nil\n}\n")))}c.isMDXComponent=!0}}]);