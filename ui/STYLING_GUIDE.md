# Talos Operator UI Styling Guide

## Overview
The Talos Operator UI uses a custom styling system based on Bootstrap with Talos-specific branding and enhancements.

## Color Palette

### Primary Colors
- **Talos Primary (Orange)**: `#FF6B35` - Used for main actions and accents
- **Talos Primary Dark**: `#E5501B` - Hover states
- **Talos Primary Light**: `#FF8C66` - Light accents

### Secondary Colors
- **Talos Secondary (Blue)**: `#004E89` - Headers and secondary actions
- **Talos Secondary Dark**: `#003A66` - Dark backgrounds
- **Talos Secondary Light**: `#1A6FA8` - Light backgrounds

### Accent & Status Colors
- **Talos Accent**: `#00A5CF` - Links and highlights
- **Success**: `#2ECC71` - Success states
- **Warning**: `#F39C12` - Warning states
- **Danger**: `#E74C3C` - Error states

### Neutral Colors
- **Dark**: `#1A1A2E` - Text
- **Light**: `#F8F9FA` - Backgrounds
- **Gray**: `#6C757D` - Secondary text
- **Gray Light**: `#E9ECEF` - Borders
- **Code Background**: `#2C3E50` - Code/YAML display

## CSS Class Reference

### Layout Classes
- `.talos-container` - Main container with shadow and rounded corners
- `.talos-header` - Gradient header with title
- `.talos-tab-content` - Tab content wrapper

### Card Classes
- `.talos-card` - Card with shadow and hover effects
- `.talos-card .card-title` - Card title with accent border

### Form Classes
- `.talos-form-group` - Form group wrapper
- `.talos-form-label` - Styled form labels (uppercase)
- `.talos-form-control` - Styled input fields
- `.talos-form-select` - Styled select dropdowns

### Button Classes
- `.talos-btn` - Base button style
- `.talos-btn-primary` - Primary action (orange gradient)
- `.talos-btn-success` - Success action (green gradient)
- `.talos-btn-secondary` - Secondary action (blue gradient)
- `.talos-button-group` - Button group container

### Navigation Classes
- `.talos-nav` - Navigation tabs container
- `.talos-nav .nav-link` - Individual tab
- `.talos-nav .nav-link.active` - Active tab

### Display Classes
- `.talos-yaml-display` - Dark theme code/YAML display
- `.talos-divider` - Section divider
- `.talos-section-title` - Section title with accent

### Utility Classes
- `.talos-animate` - Fade-in animation
- `.talos-visualizer` - Cluster visualizer styling
- `.talos-modal` - Modal dialog styling
- `.talos-toast` - Toast notification styling

## Customization

### Changing Colors
Edit CSS variables in `TalosUI.css`:
```css
:root {
  --talos-primary: #FF6B35;
  --talos-secondary: #004E89;
  /* ... other variables */
}
```

### Adding New Styles
Follow the naming convention:
- Prefix all custom classes with `.talos-`
- Use descriptive names (e.g., `.talos-form-control`, not `.tfc`)
- Group related styles together

### Responsive Breakpoints
The UI uses standard Bootstrap breakpoints:
- Mobile: < 768px
- Tablet: 768px - 991px
- Desktop: ≥ 992px

## Best Practices

1. **Always use Talos classes** instead of inline styles
2. **Maintain consistency** with existing color palette
3. **Test responsiveness** on multiple screen sizes
4. **Verify accessibility** (contrast ratios, focus states)
5. **Keep animations subtle** (0.3s transitions)

## File Structure
```
ui/src/
├── TalosUI.css              # Main custom stylesheet
├── index.css                # Global styles
├── App.css                  # App-level styles
└── components/
    ├── TalosResourceForm.tsx   # Main form component
    └── ClusterVisualizer.tsx   # Visualizer component
```

## Browser Support
- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile browsers: iOS Safari, Chrome Android

## Development Workflow

### Starting Dev Server
```bash
cd ui
npm start
```

### Building for Production
```bash
cd ui
npm run build
```

### Linting
The project uses ESLint. Warnings should be fixed before committing.

## Troubleshooting

### Styles not applying
1. Check that `TalosUI.css` is imported in the component
2. Verify class names are correct (case-sensitive)
3. Clear browser cache and reload

### Responsive issues
1. Use browser dev tools responsive mode
2. Test at breakpoints (320px, 768px, 1024px, 1440px)
3. Check for hardcoded widths/heights

### Color inconsistencies
1. Always use CSS variables, not hardcoded colors
2. Verify color accessibility with contrast checker
3. Test in light and dark mode if applicable

## Contributing
When adding new styles:
1. Follow existing naming conventions
2. Add comments for complex styles
3. Test on all supported browsers
4. Update this guide if needed
5. Keep styles maintainable and DRY

## Resources
- [Bootstrap 5 Documentation](https://getbootstrap.com/docs/5.3/)
- [React Bootstrap Components](https://react-bootstrap.github.io/)
- [CSS Best Practices](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_best_practices)
