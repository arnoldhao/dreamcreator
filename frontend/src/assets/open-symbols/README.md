Place open-symbols SVG files in this folder.

Notes
- File names should match the icon names you want to reference, e.g.:
  - gearshape.svg
  - arrow.clockwise.svg
  - magnifyingglass.svg
  - square.and.arrow.down.svg
- The app renders icons via components/base/Icon.vue using entries
  declared in src/icons/registry.js (semantic name -> file name). It
  inlines the SVG and forces fill/stroke to currentColor.
- No fallback to third-party icon libs. If an icon is missing,
  the app shows a simple placeholder and logs a console warning
  in development mode.

Static usage (optional)
- With Vite configured, you can also use static components like:
  <i-os:gearshape class="w-4 h-4" />
