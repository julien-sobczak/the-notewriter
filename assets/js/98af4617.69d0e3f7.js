"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[6641],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>k});var r=n(7294);function a(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function i(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function l(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?i(Object(n),!0).forEach((function(t){a(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):i(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function d(e,t){if(null==e)return{};var n,r,a=function(e,t){if(null==e)return{};var n,r,a={},i=Object.keys(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||(a[n]=e[n]);return a}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(r=0;r<i.length;r++)n=i[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(a[n]=e[n])}return a}var o=r.createContext({}),m=function(e){var t=r.useContext(o),n=t;return e&&(n="function"==typeof e?e(t):l(l({},t),e)),n},p=function(e){var t=m(e.components);return r.createElement(o.Provider,{value:t},e.children)},s="mdxType",u={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},c=r.forwardRef((function(e,t){var n=e.components,a=e.mdxType,i=e.originalType,o=e.parentName,p=d(e,["components","mdxType","originalType","parentName"]),s=m(n),c=a,k=s["".concat(o,".").concat(c)]||s[c]||u[c]||i;return n?r.createElement(k,l(l({ref:t},p),{},{components:n})):r.createElement(k,l({ref:t},p))}));function k(e,t){var n=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var i=n.length,l=new Array(i);l[0]=c;var d={};for(var o in t)hasOwnProperty.call(t,o)&&(d[o]=t[o]);d.originalType=e,d[s]="string"==typeof e?e:a,l[1]=d;for(var m=2;m<i;m++)l[m]=n[m];return r.createElement.apply(null,l)}return r.createElement.apply(null,n)}c.displayName="MDXCreateElement"},6364:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>o,contentTitle:()=>l,default:()=>u,frontMatter:()=>i,metadata:()=>d,toc:()=>m});var r=n(7462),a=(n(7294),n(3905));const i={sidebar_position:7},l="Reminders",d={unversionedId:"guides/reminders",id:"guides/reminders",title:"Reminders",description:"Reminders are special tags that determine a timestamp when a note must be reviewed.",source:"@site/docs/guides/reminders.md",sourceDirName:"guides",slug:"/guides/reminders",permalink:"/the-notewriter/docs/guides/reminders",draft:!1,editUrl:"https://github.com/julien-sobczak/the-notewriter/tree/main/website/docs/guides/reminders.md",tags:[],version:"current",sidebarPosition:7,frontMatter:{sidebar_position:7},sidebar:"documentationSidebar",previous:{title:"Links",permalink:"/the-notewriter/docs/guides/links"},next:{title:"Remote",permalink:"/the-notewriter/docs/guides/remote"}},o={},m=[{value:"Syntax",id:"syntax",level:2},{value:"Examples",id:"examples",level:2}],p={toc:m},s="wrapper";function u(e){let{components:t,...n}=e;return(0,a.kt)(s,(0,r.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,a.kt)("h1",{id:"reminders"},"Reminders"),(0,a.kt)("p",null,"Reminders are special tags that determine a timestamp when a note must be reviewed."),(0,a.kt)("p",null,"Reminders are displayed when planning your day using the commands ",(0,a.kt)("inlineCode",{parentName:"p"},"nt bye")," and ",(0,a.kt)("inlineCode",{parentName:"p"},"nt hi"),"."),(0,a.kt)("h2",{id:"syntax"},"Syntax"),(0,a.kt)("p",null,"The syntax must follow ",(0,a.kt)("inlineCode",{parentName:"p"},"#reminder-{expr}"),". Recurring reminders must use the additional keyword ",(0,a.kt)("inlineCode",{parentName:"p"},"every-")," like this ",(0,a.kt)("inlineCode",{parentName:"p"},"#reminder-every-{expr}"),"."),(0,a.kt)("h2",{id:"examples"},"Examples"),(0,a.kt)("admonition",{type:"info"},(0,a.kt)("p",{parentName:"admonition"},"Timestamps are always relative. For this documentation, we consider today is 2023, January 1.")),(0,a.kt)("table",null,(0,a.kt)("thead",{parentName:"table"},(0,a.kt)("tr",{parentName:"thead"},(0,a.kt)("th",{parentName:"tr",align:null},"Tag"),(0,a.kt)("th",{parentName:"tr",align:null},"Description"),(0,a.kt)("th",{parentName:"tr",align:null},"Next Occurrence(s)"))),(0,a.kt)("tbody",{parentName:"table"},(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-2023-02-01")),(0,a.kt)("td",{parentName:"tr",align:null},"Static date"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2023-02-01"))),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-every-${year}-02-01")),(0,a.kt)("td",{parentName:"tr",align:null},"Same date every year"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2023-02-01"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2024-02-01"),", ...")),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-${even-year}-02-01")),(0,a.kt)("td",{parentName:"tr",align:null},"Same date every even year"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2023-02-01"))),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-${odd-year}-02-01")),(0,a.kt)("td",{parentName:"tr",align:null},"Same date every odd year"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2024-02-01"))),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-every-2025-${month}-02")),(0,a.kt)("td",{parentName:"tr",align:null},"Every beginning of month in 2025"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2025-01-02"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2025-02-02"),", ..., ",(0,a.kt)("inlineCode",{parentName:"td"},"2025-12-02"))),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-every-2025-${odd-month}")),(0,a.kt)("td",{parentName:"tr",align:null},"Odd month with unspecified day"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2025-02-02"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2025-04-02"),", ..., ",(0,a.kt)("inlineCode",{parentName:"td"},"2025-12-02"))),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-every-${day}")),(0,a.kt)("td",{parentName:"tr",align:null},"Every day"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2023-01-01"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2023-01-02"),", ...")),(0,a.kt)("tr",{parentName:"tbody"},(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"#reminder-every-${tuesday}")),(0,a.kt)("td",{parentName:"tr",align:null},"Every Tuesday"),(0,a.kt)("td",{parentName:"tr",align:null},(0,a.kt)("inlineCode",{parentName:"td"},"2023-01-03"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2023-01-10"),", ",(0,a.kt)("inlineCode",{parentName:"td"},"2023-01-17"),", ...")))),(0,a.kt)("admonition",{type:"tip"},(0,a.kt)("p",{parentName:"admonition"},"Use reminders for notes only actionable in the future: places to visit with your kid, conference ticket registration, ...")))}u.isMDXComponent=!0}}]);