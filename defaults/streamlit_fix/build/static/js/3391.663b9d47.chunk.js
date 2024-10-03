"use strict";(self.webpackChunk_streamlit_app=self.webpackChunk_streamlit_app||[]).push([[3391],{139:(t,e,a)=>{a.d(e,{R:()=>s});var n=a(58878),r=a(38815),i=a(52050),o=a(90782);const s=t=>{var e;let{children:a}=t;const s=null===(e=(0,n.useContext)(i.E))||void 0===e?void 0:e();return s?(0,r.createPortal)(a,s):(0,o.jsx)(o.Fragment,{children:a})}},81580:(t,e,a)=>{a.d(e,{A:()=>l});var n=a(58878),r=a(32099),i=a.n(r);const o=(0,a(89653).A)("div",{target:"ecnfqzf0"})((()=>({"@media print":{display:"none"}})),"");var s=a(90782);const c=t=>{let{className:e,scriptRunId:a,numParticles:n,numParticleTypes:r,ParticleComponent:c}=t;return(0,s.jsx)(o,{className:e,"data-testid":e,children:i()(n).map((t=>{const e=Math.floor(Math.random()*r);return(0,s.jsx)(c,{particleType:e},a+t)}))})},l=(0,n.memo)(c)},30625:(t,e,a)=>{a.r(e),a.d(e,{NUM_FLAKES:()=>f,default:()=>k});var n=a(58878);const r=a.p+"static/media/flake-0.beded754e8024c73d9d2.png",i=a.p+"static/media/flake-1.8077dc154e0bf900aa73.png",o=a.p+"static/media/flake-2.e3f07d06933dd0e84c24.png";var s=a(81580),c=a(139),l=a(89653),d=a(60667);const m=function(t){let e=arguments.length>1&&void 0!==arguments[1]?arguments[1]:0;return Math.random()*(t-e)+e},p=(0,l.A)("img",{target:"ekdfb790"})((t=>{let{theme:e}=t;return{position:"fixed",top:"-150px",marginLeft:"-75px",zIndex:e.zIndices.balloons,left:`${m(90,10)}vw`,animationDelay:`${m(4e3)}ms`,height:"150px",width:"150px",pointerEvents:"none",animationDuration:"3000ms",animationName:d.i7`
  from {
    transform:
      translateY(0)
      rotateX(${m(360)}deg)
      rotateY(${m(360)}deg)
      rotateZ(${m(360)}deg);
  }

  to {
    transform:
      translateY(calc(100vh + ${150}px))
      rotateX(0)
      rotateY(0)
      rotateZ(0);
  }
`,animationTimingFunction:"ease-in",animationDirection:"normal",animationIterationCount:1,opacity:1}}),"");var u=a(90782);const f=100,h=[r,i,o],g=h.length,v=t=>{let{particleType:e}=t;return(0,u.jsx)(p,{src:h[e]})},x=function(t){let{scriptRunId:e}=t;return(0,u.jsx)(c.R,{children:(0,u.jsx)(s.A,{className:"stSnow","data-testid":"stSnow",scriptRunId:e,numParticleTypes:g,numParticles:f,ParticleComponent:v})})},k=(0,n.memo)(x)}}]);