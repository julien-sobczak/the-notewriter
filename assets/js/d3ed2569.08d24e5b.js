"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[2129],{3905:(e,t,n)=>{n.d(t,{Zo:()=>d,kt:()=>h});var r=n(7294);function a(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function o(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){a(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function s(e,t){if(null==e)return{};var n,r,a=function(e,t){if(null==e)return{};var n,r,a={},i=Object.keys(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||(a[n]=e[n]);return a}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(a[n]=e[n])}return a}var l=r.createContext({}),u=function(e){var t=r.useContext(l),n=t;return e&&(n="function"==typeof e?e(t):o(o({},t),e)),n},d=function(e){var t=u(e.components);return r.createElement(l.Provider,{value:t},e.children)},c="mdxType",p={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},m=r.forwardRef((function(e,t){var n=e.components,a=e.mdxType,i=e.originalType,l=e.parentName,d=s(e,["components","mdxType","originalType","parentName"]),c=u(n),m=a,h=c["".concat(l,".").concat(m)]||c[m]||p[m]||i;return n?r.createElement(h,o(o({ref:t},d),{},{components:n})):r.createElement(h,o({ref:t},d))}));function h(e,t){var n=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var i=n.length,o=new Array(i);o[0]=m;var s={};for(var l in t)hasOwnProperty.call(t,l)&&(s[l]=t[l]);s.originalType=e,s[c]="string"==typeof e?e:a,o[1]=s;for(var u=2;u<i;u++)o[u]=n[u];return r.createElement.apply(null,o)}return r.createElement.apply(null,n)}m.displayName="MDXCreateElement"},8196:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>o,default:()=>p,frontMatter:()=>i,metadata:()=>s,toc:()=>u});var r=n(7462),a=(n(7294),n(3905));const i={sidebar_position:2},o="Attributes",s={unversionedId:"guides/attributes",id:"guides/attributes",title:"Attributes",description:"Notes can be enriched with metadata using attributes.",source:"@site/docs/guides/attributes.md",sourceDirName:"guides",slug:"/guides/attributes",permalink:"/the-notewriter/docs/guides/attributes",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/guides/attributes.md",tags:[],version:"current",sidebarPosition:2,frontMatter:{sidebar_position:2},sidebar:"documentationSidebar",previous:{title:"Notes",permalink:"/the-notewriter/docs/guides/notes"},next:{title:"Hooks",permalink:"/the-notewriter/docs/guides/hooks"}},l={},u=[{value:"Syntax",id:"syntax",level:2},{value:"Tags",id:"tags",level:2},{value:"Types",id:"types",level:2}],d={toc:u},c="wrapper";function p(e){let{components:t,...n}=e;return(0,a.kt)(c,(0,r.Z)({},d,n,{components:t,mdxType:"MDXLayout"}),(0,a.kt)("h1",{id:"attributes"},"Attributes"),(0,a.kt)("p",null,"Notes can be enriched with metadata using attributes."),(0,a.kt)("h2",{id:"syntax"},"Syntax"),(0,a.kt)("p",null,"Attributes are defined using a YAML Front Matter at the top of the file (similar to ",(0,a.kt)("a",{parentName:"p",href:"https://jekyllrb.com/docs/front-matter/"},"Jekyll"),"):"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-md",metastring:"title=meditations.md",title:"meditations.md"},"---\nsource: _Meditations_\nauthor: Marcus Aurelius\n---\n\n# Notes\n\n## Quote: Memento Mori\n\nYou could leave life right now. Let that determine what you do and say and think.\n")),(0,a.kt)("p",null,"All notes inherit attributes defined in the YAML front matter (restrictions can be defined using ",(0,a.kt)("a",{parentName:"p",href:"/the-notewriter/docs/guides/linter"},"schemas"),")."),(0,a.kt)("p",null,"Attributes can also be defined in Markdown using the syntax ",(0,a.kt)("inlineCode",{parentName:"p"},"@name: value"),". The previous example can be rewritten like this:"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-md",metastring:"title=meditations.md",title:"meditations.md"},"# Notes\n\n## Quote: Memento Mori\n\n`@source: _Meditations_` `@author: Marcus Aurelius`\n\nYou could leave life right now. Let that determine what you do and say and think.\n")),(0,a.kt)("p",null,"Both syntaxes can be mixed."),(0,a.kt)("h2",{id:"tags"},"Tags"),(0,a.kt)("p",null,"Tags are defined using the attribute ",(0,a.kt)("inlineCode",{parentName:"p"},"tags"),":"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-md",metastring:"title=meditations.md",title:"meditations.md"},"---\ntags: [philosophy]\n---\n\n# Notes\n\n## Quote: Memento Mori\n\n`@source: _Meditations_` `@author: Marcus Aurelius`\n\nYou could leave life right now. Let that determine what you do and say and think.\n")),(0,a.kt)("p",null,"A short-hand syntax exists when declaring tags in Markdown. Both declarations are identical:"),(0,a.kt)("pre",null,(0,a.kt)("code",{parentName:"pre",className:"language-md",metastring:"title=meditations.md",title:"meditations.md"},"# Notes\n\n## Quote: Memento Mori\n\n`@tags: philosophy` `#philosophy`\n\nYou could leave life right now. Let that determine what you do and say and think.\n")),(0,a.kt)("h2",{id:"types"},"Types"),(0,a.kt)("p",null,"Attributes can be typed using ",(0,a.kt)("a",{parentName:"p",href:"/the-notewriter/docs/guides/linter"},"schemas"),"."))}p.isMDXComponent=!0}}]);