"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4955],{3905:(e,t,n)=>{n.d(t,{Zo:()=>c,kt:()=>k});var r=n(7294);function o(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function a(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){o(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function s(e,t){if(null==e)return{};var n,r,o=function(e,t){if(null==e)return{};var n,r,o={},i=Object.keys(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||(o[n]=e[n]);return o}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(o[n]=e[n])}return o}var l=r.createContext({}),p=function(e){var t=r.useContext(l),n=t;return e&&(n="function"==typeof e?e(t):a(a({},t),e)),n},c=function(e){var t=p(e.components);return r.createElement(l.Provider,{value:t},e.children)},d="mdxType",u={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},m=r.forwardRef((function(e,t){var n=e.components,o=e.mdxType,i=e.originalType,l=e.parentName,c=s(e,["components","mdxType","originalType","parentName"]),d=p(n),m=o,k=d["".concat(l,".").concat(m)]||d[m]||u[m]||i;return n?r.createElement(k,a(a({ref:t},c),{},{components:n})):r.createElement(k,a({ref:t},c))}));function k(e,t){var n=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var i=n.length,a=new Array(i);a[0]=m;var s={};for(var l in t)hasOwnProperty.call(t,l)&&(s[l]=t[l]);s.originalType=e,s[d]="string"==typeof e?e:o,a[1]=s;for(var p=2;p<i;p++)a[p]=n[p];return r.createElement.apply(null,a)}return r.createElement.apply(null,n)}m.displayName="MDXCreateElement"},2529:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>a,default:()=>u,frontMatter:()=>i,metadata:()=>s,toc:()=>p});var r=n(7462),o=(n(7294),n(3905));const i={sidebar_position:6},a="Links",s={unversionedId:"guides/links",id:"guides/links",title:"Links",description:"Relation Links",source:"@site/docs/guides/links.md",sourceDirName:"guides",slug:"/guides/links",permalink:"/the-notewriter/docs/guides/links",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/guides/links.md",tags:[],version:"current",sidebarPosition:6,frontMatter:{sidebar_position:6},sidebar:"documentationSidebar",previous:{title:"Flashcards",permalink:"/the-notewriter/docs/guides/flashcards"},next:{title:"Reminders",permalink:"/the-notewriter/docs/guides/reminders"}},l={},p=[{value:"Relation Links",id:"relation-links",level:2},{value:"<code>references</code>",id:"references",level:3},{value:"<code>source</code>",id:"source",level:3},{value:"<code>inspirations</code>",id:"inspirations",level:3},{value:"Go Links",id:"go-links",level:2}],c={toc:p},d="wrapper";function u(e){let{components:t,...n}=e;return(0,o.kt)(d,(0,r.Z)({},c,n,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"links"},"Links"),(0,o.kt)("h2",{id:"relation-links"},"Relation Links"),(0,o.kt)("p",null,"Notes can reference each other using wikilinks (ex: ",(0,o.kt)("inlineCode",{parentName:"p"},"[[file#note]]"),")."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md"},"## Note: A\n\nCheck note [[#A]].\n\n## Note: B\n\nCheck note [[#B]].\n")),(0,o.kt)("p",null,"Special attributes are also analyzed to determine the relations between notes."),(0,o.kt)("ul",null,(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("inlineCode",{parentName:"li"},"references")," (type: ",(0,o.kt)("inlineCode",{parentName:"li"},"array"),")"),(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("inlineCode",{parentName:"li"},"source")," (type: ",(0,o.kt)("inlineCode",{parentName:"li"},"string"),")"),(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("inlineCode",{parentName:"li"},"inspirations")," (type: ",(0,o.kt)("inlineCode",{parentName:"li"},"string"),")")),(0,o.kt)("p",null,"Wikilinks inside these attributes automatically generate relations (= links in ",(0,o.kt)("em",{parentName:"p"},"The NoteWriter Desktop"),")."),(0,o.kt)("h3",{id:"references"},(0,o.kt)("inlineCode",{parentName:"h3"},"references")),(0,o.kt)("admonition",{type:"tip"},(0,o.kt)("p",{parentName:"admonition"},"Use the ",(0,o.kt)("inlineCode",{parentName:"p"},"references")," attribute to mention that another note ",(0,o.kt)("strong",{parentName:"p"},"is referenced by")," a website, a book, or another note.")),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md"},"## Note: A\n\n`@references: https://random.website`\n`@references: _A Random Book_`\n`@references: [[#B]]`\n\nA first note.\n\n## Note B\n\nA second note.\n")),(0,o.kt)("p",null,"The last reference is similar to:"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md"},"## Note: A\n\nA first note.\n\n## Note: B\n\nA second note referencing [[#A]]\n")),(0,o.kt)("h3",{id:"source"},(0,o.kt)("inlineCode",{parentName:"h3"},"source")),(0,o.kt)("admonition",{type:"tip"},(0,o.kt)("p",{parentName:"admonition"},"Use the ",(0,o.kt)("inlineCode",{parentName:"p"},"source")," attribute to remember if a note was collected from a book, a website, etc.")),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md"},"# Note: A\n\n`@source: https://some.random.blog`\n")),(0,o.kt)("h3",{id:"inspirations"},(0,o.kt)("inlineCode",{parentName:"h3"},"inspirations")),(0,o.kt)("admonition",{type:"tip"},(0,o.kt)("p",{parentName:"admonition"},"Use the ",(0,o.kt)("inlineCode",{parentName:"p"},"inspirations")," attribute to specify which work has inspired this note (a website, a book, another note, ...)")),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md"},"## Note: A\n\n`@inspiration: [[books/book-A#Quote: On Note-Taking]]`\n\nA note.\n")),(0,o.kt)("h2",{id:"go-links"},"Go Links"),(0,o.kt)("p",null,"Markdown links can include in their title was is called a ",(0,o.kt)("em",{parentName:"p"},"Go link"),", that is a memorable name for a hard-to-remember URL."),(0,o.kt)("p",null,"The syntax must follow the convention ",(0,o.kt)("inlineCode",{parentName:"p"},"#go/{name}"),"."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-md",metastring:"title:go.md","title:go.md":!0},'## Note: Useful Links\n\n* [Golang](https://go.dev/doc/ "#go/go") was designed by Robert Greisemer, Rob Pike, and Ken Thompson at Google in 2007.\n* [Go Playground](https://go.dev/play/ "#go/go/playground") is useful to share snippets.\n')),(0,o.kt)("p",null,"Go links can be browse directly from the terminal:"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-shell"},"$ nt go go\n# Open a new tab in your browser to https://go.dev/doc/\n\n# Or\n$ nt go go/playground\n")),(0,o.kt)("p",null,"You can also use Go links (more conveniently) since ",(0,o.kt)("em",{parentName:"p"},"The NoteWriter Desktop")," (no need to have a terminal open inside your notes repository)."),(0,o.kt)("admonition",{type:"tip"},(0,o.kt)("p",{parentName:"admonition"},"Use Go links for URL that you must visit frequently (ex: internal tools at work).")))}u.isMDXComponent=!0}}]);