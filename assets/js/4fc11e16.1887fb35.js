"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[1233],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>f});var i=n(7294);function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function a(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);t&&(i=i.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,i)}return n}function o(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?a(Object(n),!0).forEach((function(t){r(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):a(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function l(e,t){if(null==e)return{};var n,i,r=function(e,t){if(null==e)return{};var n,i,r={},a=Object.keys(e);for(i=0;i<a.length;i++)n=a[i],t.indexOf(n)>=0||(r[n]=e[n]);return r}(e,t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(i=0;i<a.length;i++)n=a[i],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(r[n]=e[n])}return r}var s=i.createContext({}),d=function(e){var t=i.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):o(o({},t),e)),n},p=function(e){var t=d(e.components);return i.createElement(s.Provider,{value:t},e.children)},u="mdxType",m={inlineCode:"code",wrapper:function(e){var t=e.children;return i.createElement(i.Fragment,{},t)}},c=i.forwardRef((function(e,t){var n=e.components,r=e.mdxType,a=e.originalType,s=e.parentName,p=l(e,["components","mdxType","originalType","parentName"]),u=d(n),c=r,f=u["".concat(s,".").concat(c)]||u[c]||m[c]||a;return n?i.createElement(f,o(o({ref:t},p),{},{components:n})):i.createElement(f,o({ref:t},p))}));function f(e,t){var n=arguments,r=t&&t.mdxType;if("string"==typeof e||r){var a=n.length,o=new Array(a);o[0]=c;var l={};for(var s in t)hasOwnProperty.call(t,s)&&(l[s]=t[s]);l.originalType=e,l[u]="string"==typeof e?e:r,o[1]=l;for(var d=2;d<a;d++)o[d]=n[d];return i.createElement.apply(null,o)}return i.createElement.apply(null,n)}c.displayName="MDXCreateElement"},9236:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>s,contentTitle:()=>o,default:()=>m,frontMatter:()=>a,metadata:()=>l,toc:()=>d});var i=n(7462),r=(n(7294),n(3905));const a={sidebar_position:3},o="Medias",l={unversionedId:"guides/medias",id:"guides/medias",title:"Medias",description:"Notes can include medias (images, videos, audios) using the usual Markdown syntax.",source:"@site/docs/guides/medias.md",sourceDirName:"guides",slug:"/guides/medias",permalink:"/the-notewriter/docs/guides/medias",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/guides/medias.md",tags:[],version:"current",sidebarPosition:3,frontMatter:{sidebar_position:3},sidebar:"documentationSidebar",previous:{title:"Hooks",permalink:"/the-notewriter/docs/guides/hooks"},next:{title:"Linter",permalink:"/the-notewriter/docs/guides/linter"}},s={},d=[{value:"Conversion",id:"conversion",level:2}],p={toc:d},u="wrapper";function m(e){let{components:t,...n}=e;return(0,r.kt)(u,(0,i.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,r.kt)("h1",{id:"medias"},"Medias"),(0,r.kt)("p",null,"Notes can include medias (images, videos, audios) using the usual Markdown syntax."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-md"},"## Reference: Me\n\n![Profile](medias/me.png)\n")),(0,r.kt)("h2",{id:"conversion"},"Conversion"),(0,r.kt)("p",null,"All medias are converted using the external dependency ",(0,r.kt)("inlineCode",{parentName:"p"},"ffmpeg"),":"),(0,r.kt)("ul",null,(0,r.kt)("li",{parentName:"ul"},"Images (",(0,r.kt)("inlineCode",{parentName:"li"},"jpeg"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"png"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"gif"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"tiff"),", ...) \u27a1\ufe0f ",(0,r.kt)("inlineCode",{parentName:"li"},"avif"),(0,r.kt)("ul",{parentName:"li"},(0,r.kt)("li",{parentName:"ul"},"A thumbnail image is generated (useful when displaying a list of notes)"),(0,r.kt)("li",{parentName:"ul"},"A medium image is generated (useful when displaying a single note)"))),(0,r.kt)("li",{parentName:"ul"},"Audios (",(0,r.kt)("inlineCode",{parentName:"li"},"wav"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"aac"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"flac"),", ...) \u27a1\ufe0f ",(0,r.kt)("inlineCode",{parentName:"li"},"mp3"),(0,r.kt)("ul",{parentName:"li"},(0,r.kt)("li",{parentName:"ul"},"A single audio is generated from the original file."))),(0,r.kt)("li",{parentName:"ul"},"Videos (",(0,r.kt)("inlineCode",{parentName:"li"},"mp4"),", ",(0,r.kt)("inlineCode",{parentName:"li"},"avi"),", ...) \u27a1\ufe0f ",(0,r.kt)("inlineCode",{parentName:"li"},"webm"),(0,r.kt)("ul",{parentName:"li"},(0,r.kt)("li",{parentName:"ul"},"A ",(0,r.kt)("inlineCode",{parentName:"li"},"avif")," image is generated using the first frame.")))),(0,r.kt)("p",null,"Original files are not used directly (= not stored in ",(0,r.kt)("inlineCode",{parentName:"p"},".nt/objects"),"). The applications ",(0,r.kt)("em",{parentName:"p"},"The NoteWriter Desktop")," and ",(0,r.kt)("em",{parentName:"p"},"The NoteWriter Nomad")," rely on optimized versions to reduce the storage and network bandwidth requirements."),(0,r.kt)("admonition",{type:"tip"},(0,r.kt)("p",{parentName:"admonition"},"Place your medias in a ",(0,r.kt)("inlineCode",{parentName:"p"},"medias/")," directory present along your note file to navigate easily in your editor.")))}m.isMDXComponent=!0}}]);