import { createApp, h, reactive } from 'vue'

// color dot per type (8px circle)
function dotStyleByType(type) {
  const base = { width:'8px', height:'8px', borderRadius:'50%', flexShrink:0 }
  switch (type) {
    case 'success': return { ...base, background:'#30d158' }
    case 'error': return { ...base, background:'#ff453a' }
    case 'warning': return { ...base, background:'#ff9f0a' }
    default: return { ...base, background:'rgba(60,60,67,0.35)' }
  }
}

// Public setup function used by app (macOS-styled UI)
export function setupMacUI() {
  setupMacosMessage()
  setupMacosNotification()
  setupMacosDialog()
}

// Message (HUD, top-center)
function setupMacosMessage() {
  const queue = reactive([])
  let defaultMaxWidth = '420px'
  const MacContainer = {
    name: 'MacToastContainer',
    setup() {
      return () => h('div', {
        id: 'macos-toast-root',
        style: {
          position: 'fixed', inset: 0, zIndex: 2147483647, pointerEvents: 'none',
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', paddingTop: '44px'
        }
      }, queue.map(item => h(MacToast, { key: item.id, ...item })))
    }
  }
  const MacToast = {
    name: 'MacToast', props: { id:String, type:{type:String,default:'info'}, content:String, duration:{type:Number,default:2600}, maxWidth:[String,Number] },
    setup(props) {
      const remove = () => { const i=queue.findIndex(q=>q.id===props.id); if(i>=0) queue.splice(i,1) }
      setTimeout(remove, props.duration)
      const pal = props.type==='success'? { border:'rgba(52,199,89,0.45)', text:'#1d1d1f', icon:'#34c759' }
        : props.type==='error'? { border:'rgba(255,69,58,0.45)', text:'#1d1d1f', icon:'#ff453a' }
        : props.type==='warning'? { border:'rgba(255,159,10,0.45)', text:'#1d1d1f', icon:'#ff9f0a' }
        : { border:'rgba(60,60,67,0.2)', text:'#1d1d1f', icon:'rgba(60,60,67,0.6)' }
      const maxW = typeof props.maxWidth === 'number' ? (props.maxWidth + 'px') : (props.maxWidth || '420px')
      return () => h('div', { style: {
        pointerEvents:'none', backdropFilter:'saturate(180%) blur(22px)', WebkitBackdropFilter:'saturate(180%) blur(22px)',
        background:'rgba(245,245,247,0.72)', border:'1px solid '+pal.border, color:pal.text,
        boxShadow:'0 6px 18px rgba(0,0,0,0.18)', borderRadius:'12px', padding:'8px 12px',
        maxWidth: maxW,
        fontSize:'13px', lineHeight:'1.2', display:'flex', alignItems:'center', gap:'8px'
      } }, [
        h('div', { style: dotStyleByType(props.type) }),
        // 内容区域：短文本自适应，长文本在最大宽度内换行
        h('div', { style:{
          fontSize:'12px',
          whiteSpace:'pre-wrap',
          overflowWrap:'anywhere',
          wordBreak:'break-word',
          minWidth: 0
        } }, props.content || '')
      ])
    }
  }
  function mount(){ if(document.getElementById('macos-toast-root')) return; const host=document.createElement('div'); document.body.appendChild(host); createApp(MacContainer).mount(host) }
  function push(content, options={}){
    mount()
    const id=Date.now()+'_'+Math.random().toString(36).slice(2,7)
    const type=options.type||'info'
    const duration=options.duration!=null?options.duration:2600
    const maxWidth = options.maxWidth != null ? options.maxWidth : defaultMaxWidth
    queue.push({ id, content, type, duration, maxWidth })
    return { close(){ const i=queue.findIndex(q=>q.id===id); if(i>=0) queue.splice(i,1) } }
  }
  window.$message = {
    success(c,o){return push(c,{...o,type:'success'})},
    error(c,o){return push(c,{...o,type:'error'})},
    warning(c,o){return push(c,{...o,type:'warning'})},
    info(c,o){return push(c,{...o,type:'info'})},
    config(opts){
      if (!opts) return
      if (opts.maxWidth != null) {
        defaultMaxWidth = typeof opts.maxWidth === 'number' ? (opts.maxWidth + 'px') : String(opts.maxWidth)
      }
    }
  }
}

// Notification (top-right cards)
function setupMacosNotification(){
  const queue = reactive([])
  const NotiContainer = { name:'MacNotiContainer', setup(){ return () => h('div', { id:'macos-notify-root', style:{ position:'fixed', inset:0, zIndex:2147483630, pointerEvents:'none' } }, [ h('div', { style:{ position:'absolute', top:'16px', right:'16px', display:'flex', flexDirection:'column', gap:'10px', pointerEvents:'none' } }, queue.map(item=>h(NotiCard,{ key:item.id, ...item }))) ]) } }
  const NotiCard = { name:'MacNotiCard', props:{ id:String, title:String, content:[String,Object], type:{type:String,default:'info'}, duration:{type:Number,default:4500}, closable:{type:Boolean,default:true}, action:[Function,Object] }, setup(props){
    const remove=()=>{ const i=queue.findIndex(n=>n.id===props.id); if(i>=0) queue.splice(i,1) }
    if(props.duration>0) setTimeout(remove, props.duration)
    const pal = props.type==='success'? { border:'rgba(52,199,89,0.45)', text:'#1d1d1f', icon:'#34c759' }
      : props.type==='error'? { border:'rgba(255,69,58,0.45)', text:'#1d1d1f', icon:'#ff453a' }
      : props.type==='warning'? { border:'rgba(255,159,10,0.45)', text:'#1d1d1f', icon:'#ff9f0a' }
      : { border:'rgba(60,60,67,0.2)', text:'#1d1d1f', icon:'rgba(60,60,67,0.6)' }
    return () => h('div', { style:{ pointerEvents:'auto', backdropFilter:'saturate(180%) blur(22px)', WebkitBackdropFilter:'saturate(180%) blur(22px)', background:'rgba(245,245,247,0.8)', border:'1px solid '+pal.border, color:pal.text, boxShadow:'0 8px 22px rgba(0,0,0,0.20)', borderRadius:'12px', padding:'10px 12px', minWidth:'300px', maxWidth:'420px' } }, [
      h('div', { style:{ display:'flex', alignItems:'center', gap:'10px' } }, [
        h('div', { style: dotStyleByType(props.type) }),
        h('div', { style:{ flex:1, minWidth:0 } }, [
          props.title ? h('div', { style:{ fontWeight:600, fontSize:'12px', lineHeight:1.2, marginBottom: props.content ? '4px':'0' } }, props.title) : null,
          props.content ? h('div', { style:{ fontSize:'12px', lineHeight:1.35, whiteSpace:'pre-wrap' } }, props.content) : null,
          props.action ? h('div', { style:{ display:'flex', justifyContent:'flex-end', gap:'8px', marginTop:'8px' } }, [ typeof props.action === 'function' ? props.action(remove) : props.action ]) : null
        ]),
        props.closable ? h('button', { style:{ width:'22px', height:'22px', borderRadius:'6px', border:'none', background:'transparent', color:'rgba(60,60,67,0.6)', cursor:'pointer' }, onClick: remove, title:'Close' }, [ h('svg', { xmlns:'http://www.w3.org/2000/svg', viewBox:'0 0 24 24', width:'14', height:'14' }, [ h('path', { d:'M6 6L18 18M6 18L18 6', stroke:'currentColor', 'stroke-width':'2', 'stroke-linecap':'round' }) ]) ]) : null
      ])
    ]) }
  }
  function mount(){ if(document.getElementById('macos-notify-root')) return; const host=document.createElement('div'); document.body.appendChild(host); createApp(NotiContainer).mount(host) }
  function push(option){ mount(); const id=Date.now()+'_'+Math.random().toString(36).slice(2,7); const entry=Object.assign({ id, type:'info', duration:4500, closable:true }, option); queue.push(entry); return { destroy(){ const i=queue.findIndex(n=>n.id===id); if(i>=0) queue.splice(i,1) }, close(){ this.destroy() } } }
  window.$notification = { create(c,o={}){ return push(typeof c==='string'? Object.assign({ content:c }, o): (c||{})) }, show(c,o={}){ return this.create(c,o) }, success(c,o={}){ return this.create(typeof c==='string'? c : Object.assign({}, c, { type:'success' }), typeof c==='string'? Object.assign({}, o, { type:'success' }): {} ) }, error(c,o={}){ return this.create(typeof c==='string'? c : Object.assign({}, c, { type:'error' }), typeof c==='string'? Object.assign({}, o, { type:'error' }): {} ) }, warning(c,o={}){ return this.create(typeof c==='string'? c : Object.assign({}, c, { type:'warning' }), typeof c==='string'? Object.assign({}, o, { type:'warning' }): {} ) }, info(c,o={}){ return this.create(typeof c==='string'? c : Object.assign({}, c, { type:'info' }), typeof c==='string'? Object.assign({}, o, { type:'info' }): {} ) } }
}

// Dialog (centered alert)
function setupMacosDialog(){
  const stack = reactive([])
  const Host = { name:'MacDialogHost', setup(){ return () => h('div', { id:'macos-dialog-root', style:{ position:'fixed', inset:0, zIndex:2147483646, pointerEvents: stack.length ? 'auto':'none' } }, stack.map(d=>h(Card,{ key:d.id, ...d }))) } }
  const Card = { name:'MacDialogCard', props:{ id:String, title:String, content:[String,Object], type:{type:String,default:'info'}, closable:{type:Boolean,default:true}, positiveText:String, negativeText:String, onPositiveClick:Function, onNegativeClick:Function, onClose:Function }, setup(props){
    const remove=()=>{ const i=stack.findIndex(x=>x.id===props.id); if(i>=0) stack.splice(i,1); props.onClose&&props.onClose() }
    const pos=()=>{ props.onPositiveClick&&props.onPositiveClick(); remove() }
    const neg=()=>{ props.onNegativeClick&&props.onNegativeClick(); remove() }
    const iconColor = props.type==='success'? '#34c759' : props.type==='error'? '#ff453a' : props.type==='warning'? '#ff9f0a' : 'rgba(60,60,67,0.6)'
    return () => h('div', { style:{ position:'fixed', inset:0, display:'flex', alignItems:'center', justifyContent:'center', pointerEvents:'auto' } }, [
      h('div', { style:{ position:'absolute', inset:0, background:'rgba(0,0,0,0.2)', backdropFilter:'blur(4px)' }, onClick: props.closable? remove:null }),
      h('div', { style:{ position:'relative', zIndex:1, width:'min(420px,92vw)', background:'rgba(245,245,247,0.9)', backdropFilter:'saturate(180%) blur(22px)', border:'1px solid rgba(60,60,67,0.2)', color:'#1d1d1f', borderRadius:'12px', boxShadow:'0 18px 48px rgba(0,0,0,0.25)' } }, [
        h('div', { style:{ padding:'14px 16px', display:'flex', alignItems:'center', gap:'10px', borderBottom:'1px solid rgba(60,60,67,0.15)' } }, [
          h('div', { style: dotStyleByType(props.type) }),
          h('div', { style:{ fontWeight:600, fontSize:'12px', lineHeight:1.2 } }, props.title||'Info')
        ]),
        h('div', { style:{ padding:'12px 16px', fontSize:'13px', lineHeight:1.35, whiteSpace:'pre-wrap' } }, props.content||''),
        h('div', { style:{ padding:'10px 12px 12px', display:'flex', alignItems:'center', justifyContent:'flex-end', gap:'10px', borderTop:'1px solid rgba(60,60,67,0.15)' } }, [
          props.negativeText ? h('button', { class: 'btn-chip-ghost', onClick:neg }, props.negativeText) : null,
          h('button', { class: 'btn-chip btn-primary', onClick:pos }, props.positiveText||'OK')
        ])
      ])
    ]) }
  }
  // buttons now use global btn-chip family styles
  function mount(){ if(document.getElementById('macos-dialog-root')) return; const host=document.createElement('div'); document.body.appendChild(host); createApp(Host).mount(host) }
  function push(opt){ mount(); const id=Date.now()+'_'+Math.random().toString(36).slice(2,7); const entry=Object.assign({ id, type:'info', closable:true, positiveText:'OK' }, opt); stack.push(entry); return { destroy(){ const i=stack.findIndex(x=>x.id===id); if(i>=0) stack.splice(i,1) }, close(){ this.destroy() } } }
  window.$dialog = { success(c,o={}){ return push(typeof c==='string'? { title:'Success', content:c, type:'success', positiveText:o.positiveText } : Object.assign({}, c, { type:'success' })) }, error(c,o={}){ return push(typeof c==='string'? { title:'Error', content:c, type:'error', positiveText:o.positiveText } : Object.assign({}, c, { type:'error' })) }, warning(c,o={}){ return push(typeof c==='string'? { title:'Warning', content:c, type:'warning', positiveText:o.positiveText } : Object.assign({}, c, { type:'warning' })) }, info(c,o={}){ return push(typeof c==='string'? { title:'Info', content:c, type:'info', positiveText:o.positiveText } : Object.assign({}, c, { type:'info' })) }, confirm(c,o={}){ return push({ title:o.title||'Confirm', content: typeof c==='string'? c : (c.content||''), type:'info', positiveText:o.positiveText||'OK', negativeText:o.negativeText||'Cancel', onPositiveClick:o.onPositiveClick, onNegativeClick:o.onNegativeClick }) } }
}
