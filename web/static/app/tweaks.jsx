// Tweaks panel

const Tweaks = ({ theme, setTheme, accent, setAccent, density, setDensity, onClose }) => {
  return (
    <div className="tweaks-panel">
      <div className="tweaks-head">
        <h4>Tweaks</h4>
        <button className="icon-btn" onClick={onClose} style={{width:24, height:24}}><IconClose size={13}/></button>
      </div>
      <div className="tweaks-body">
        <div>
          <div className="tweaks-section-label">Theme</div>
          <div className="theme-row">
            <div className={`theme-opt ${theme==='dark'?'selected':''}`} onClick={()=>setTheme('dark')}>Dark</div>
            <div className={`theme-opt ${theme==='light'?'selected':''}`} onClick={()=>setTheme('light')}>Light</div>
          </div>
        </div>
        <div>
          <div className="tweaks-section-label">Accent</div>
          <div className="swatch-row">
            {[
              { id: 'indigo', color: 'oklch(0.58 0.18 270)' },
              { id: 'violet', color: 'oklch(0.58 0.2 300)' },
              { id: 'teal', color: 'oklch(0.62 0.12 195)' },
            ].map(s => (
              <div key={s.id} className={`swatch ${accent===s.id?'selected':''}`} style={{background: s.color}} onClick={()=>setAccent(s.id)} title={s.id}/>
            ))}
          </div>
        </div>
        <div>
          <div className="tweaks-section-label">Density</div>
          <div className="theme-row">
            <div className={`theme-opt ${density==='compact'?'selected':''}`} onClick={()=>setDensity('compact')}>Compact</div>
            <div className={`theme-opt ${density==='balanced'?'selected':''}`} onClick={()=>setDensity('balanced')}>Balanced</div>
            <div className={`theme-opt ${density==='spacious'?'selected':''}`} onClick={()=>setDensity('spacious')}>Spacious</div>
          </div>
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { Tweaks });
