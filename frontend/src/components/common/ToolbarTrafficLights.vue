<script setup>
import { Quit, WindowMinimise, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'

const handleClose = () => { try { Quit() } catch {} }
const handleMinimise = () => { try { WindowMinimise() } catch {} }
const handleMaximise = () => { try { WindowToggleMaximise() } catch {} }
</script>

<template>
  <div class="toolbar-traffic" aria-label="Window controls">
    <button class="tl tl-close" aria-label="Close" @click.stop="handleClose">
      <span class="dot dot-red"></span>
    </button>
    <button class="tl tl-min" aria-label="Minimise" @click.stop="handleMinimise">
      <span class="dot dot-yellow disabled"></span>
    </button>
    <button class="tl tl-max" aria-label="Maximise" @click.stop="handleMaximise">
      <span class="dot dot-green disabled"></span>
    </button>
  </div>
</template>

<style scoped>
.toolbar-traffic {
  display: flex;
  align-items: center;
  gap: 6px;
  /* Prevent dragging when clicking controls inside the draggable titlebar */
  -webkit-app-region: no-drag;
  --wails-draggable: no-drag;
  /* theme tokens for lights */
  --tl-red: #ff5f56;
  --tl-yellow: #ffbd2e;
  --tl-green: #28c940;
  --tl-ring: rgba(0,0,0,0.12);
}
.tl {
  border: 0;
  padding: 0;
  background: transparent;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  cursor: default;
}
.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
  position: relative;
  box-shadow: inset 0 0 0 1px var(--tl-ring);
  transition: background-color 120ms ease, box-shadow 120ms ease, transform 60ms ease, filter 120ms ease, opacity 120ms ease;
}
.tl-close .dot { background: var(--tl-red); }

/* grey by default, mac-like color on hover (yellow/green only) */
.tl-min .dot { background: #d9d9d9; }
.tl-min:hover .dot { background: var(--tl-yellow); }

.tl-max .dot { background: #d9d9d9; }
.tl-max:hover .dot { background: var(--tl-green); }

/* vector glyphs centered without fonts */
.tl .dot::before,
.tl .dot::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  background: rgba(0,0,0,0.65);
  border-radius: 1px;
  opacity: 0;
  transition: opacity 120ms ease;
  pointer-events: none;
}
.tl:hover .dot::before,
.tl:hover .dot::after { opacity: 0.9; }

/* close: two diagonal bars */
.tl-close .dot::before { width: 6px; height: 1px; transform: translate(-50%, -50%) rotate(45deg); }
.tl-close .dot::after  { width: 6px; height: 1px; transform: translate(-50%, -50%) rotate(-45deg); }

/* minimise: single horizontal bar */
.tl-min .dot::after { width: 6px; height: 1px; transform: translate(-50%, -50%); }

/* maximise: horizontal + vertical bars */
.tl-max .dot::after { width: 6px; height: 1px; transform: translate(-50%, -50%); }
.tl-max .dot::before { width: 1px; height: 6px; transform: translate(-50%, -50%); }

/* subtle ring/press feedback */
.tl:hover .dot { box-shadow: inset 0 0 0 1px rgba(0,0,0,0.18), inset 0 1px 0 rgba(255,255,255,0.35); }
.tl:active .dot { filter: brightness(0.95); transform: scale(0.96); box-shadow: inset 0 0 0 1px rgba(0,0,0,0.22), inset 0 1px 0 rgba(255,255,255,0.3); }

.disabled { opacity: .95; }
</style>
