"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4183],{3905:(e,t,r)=>{r.d(t,{Zo:()=>l,kt:()=>f});var n=r(7294);function o(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function i(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function a(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?i(Object(r),!0).forEach((function(t){o(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):i(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function s(e,t){if(null==e)return{};var r,n,o=function(e,t){if(null==e)return{};var r,n,o={},i=Object.keys(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||(o[r]=e[r]);return o}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(o[r]=e[r])}return o}var c=n.createContext({}),p=function(e){var t=n.useContext(c),r=t;return e&&(r="function"==typeof e?e(t):a(a({},t),e)),r},l=function(e){var t=p(e.components);return n.createElement(c.Provider,{value:t},e.children)},m="mdxType",d={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},u=n.forwardRef((function(e,t){var r=e.components,o=e.mdxType,i=e.originalType,c=e.parentName,l=s(e,["components","mdxType","originalType","parentName"]),m=p(r),u=o,f=m["".concat(c,".").concat(u)]||m[u]||d[u]||i;return r?n.createElement(f,a(a({ref:t},l),{},{components:r})):n.createElement(f,a({ref:t},l))}));function f(e,t){var r=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var i=r.length,a=new Array(i);a[0]=u;var s={};for(var c in t)hasOwnProperty.call(t,c)&&(s[c]=t[c]);s.originalType=e,s[m]="string"==typeof e?e:o,a[1]=s;for(var p=2;p<i;p++)a[p]=r[p];return n.createElement.apply(null,a)}return n.createElement.apply(null,r)}u.displayName="MDXCreateElement"},556:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>c,contentTitle:()=>a,default:()=>d,frontMatter:()=>i,metadata:()=>s,toc:()=>p});var n=r(7462),o=(r(7294),r(3905));const i={sidebar_position:2},a="Editing Notes with VS Code",s={unversionedId:"practices/vs-code",id:"practices/vs-code",title:"Editing Notes with VS Code",description:"The NoteWriter works with any editor. If you are using VS Code, this page contains my personal tips.",source:"@site/docs/practices/vs-code.md",sourceDirName:"practices",slug:"/practices/vs-code",permalink:"/the-notewriter/docs/practices/vs-code",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/practices/vs-code.md",tags:[],version:"current",sidebarPosition:2,frontMatter:{sidebar_position:2},sidebar:"documentationSidebar",previous:{title:"Guidelines",permalink:"/the-notewriter/docs/practices/guidelines"},next:{title:"My Workflow",permalink:"/the-notewriter/docs/practices/my-workflow"}},c={},p=[{value:"Recommended Snippets",id:"recommended-snippets",level:2},{value:"Recommended Plugins",id:"recommended-plugins",level:2}],l={toc:p},m="wrapper";function d(e){let{components:t,...r}=e;return(0,o.kt)(m,(0,n.Z)({},l,r,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"editing-notes-with-vs-code"},"Editing Notes with VS Code"),(0,o.kt)("p",null,(0,o.kt)("em",{parentName:"p"},"The NoteWriter")," works with any editor. If you are using VS Code, this page contains my personal tips."),(0,o.kt)("h2",{id:"recommended-snippets"},"Recommended Snippets"),(0,o.kt)("p",null,(0,o.kt)("strong",{parentName:"p"},"TODO")," complete"),(0,o.kt)("h2",{id:"recommended-plugins"},"Recommended Plugins"),(0,o.kt)("ul",null,(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("a",{parentName:"li",href:"https://foambubble.github.io/foam/"},"Foam"),": Great list of ",(0,o.kt)("a",{parentName:"li",href:"https://foambubble.github.io/foam/user/getting-started/recommended-extensions"},"plugins")," to work with Markdown files."),(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("a",{parentName:"li",href:"https://marketplace.visualstudio.com/items?itemName=bierner.emojisense"},":emojisense:"),", by Matt Bierner: Enter emojis faster using autocompletion. Rely on ",(0,o.kt)("inlineCode",{parentName:"li"},"github/gemoji")," (see ",(0,o.kt)("a",{parentName:"li",href:"https://github.com/github/gemoji/blob/master/db/emoji.json"},"complete listing"),")."),(0,o.kt)("li",{parentName:"ul"},(0,o.kt)("a",{parentName:"li",href:"https://www.grammarly.com/"},"Grammarly"),': Great to catch most typos and grammar errors if you accept to have "errors" with arcane terminology. Here is the plugin configuration (cf ',(0,o.kt)("inlineCode",{parentName:"li"},".vscode/settings.json"),") to limit Grammarly on Markdown files:",(0,o.kt)("pre",{parentName:"li"},(0,o.kt)("code",{parentName:"pre",className:"language-json"},'{\n    "grammarly.selectors": [\n        {\n            "language": "markdown",\n            "scheme": "file"\n        }\n    ]\n}\n')))),(0,o.kt)("admonition",{type:"tip"},(0,o.kt)("p",{parentName:"admonition"},(0,o.kt)("em",{parentName:"p"},"How to enable the extension on specific workspaces only?")),(0,o.kt)("p",{parentName:"admonition"},"You may not want to run Grammarly on every workspace on your laptop. ",(0,o.kt)("a",{parentName:"p",href:"https://github.com/microsoft/vscode/issues/15611"},"VS Code supports disabling it globally and enabling it specifically on a few workspaces"),".")))}d.isMDXComponent=!0}}]);