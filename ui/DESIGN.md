# Design System Documentation: Tactical Precision & Cinematic Depth

## 1. Overview & Creative North Star: "The Obsidian Sentinel"
This design system is built to evoke the high-stakes, quiet authority of a modern command center. We are moving away from the cluttered, "hacker-green" terminal aesthetic of the 90s and toward a sophisticated, editorial interface that prioritizes clarity, tactical depth, and cinematic immersion.

**The Creative North Star: The Obsidian Sentinel.** 
The interface should feel like a single, seamless piece of glass. We achieve this by rejecting rigid grids and harsh dividers in favor of **Intentional Asymmetry** and **Tonal Layering**. Data is not just displayed; it is curated. High-contrast typography scales and overlapping translucent surfaces create a sense of physical space within the digital screen, ensuring the operator feels in total control of the simulation.

---

## 2. Colors: Obsidian Depth & Emerald Luminescence
The palette is rooted in the "Deep Obsidian" spectrum, using subtle shifts in dark values to define UI regions rather than structural lines.

### The Palette (Material Design Tokens)
*   **Background:** `#131313` (The base void)
*   **Primary (Accent):** `#4edea3` (The Emerald glow)
*   **Primary Container:** `#10b981` (The functional emerald)
*   **Surface Tiers:**
    *   `surface_container_lowest`: `#0e0e0e` (Deepest insets)
    *   `surface_container_low`: `#1c1b1b` (Standard secondary panels)
    *   `surface_container_highest`: `#353534` (Elevated floating elements)

### The "No-Line" Rule
**Explicit Instruction:** Do not use 1px solid borders to section off the UI. Standard borders feel like "web templates." Instead, define boundaries through:
1.  **Background Shifts:** Place a `surface_container_low` panel directly against the `surface` background.
2.  **Negative Space:** Use the Spacing Scale (specifically `12` or `16` tokens) to create architectural "voids" between functional groups.

### Surface Hierarchy & Nesting
Treat the UI as stacked sheets of frosted obsidian. 
*   **Base:** `surface` (#131313).
*   **Primary Workspace:** `surface_container_low`.
*   **Contextual Overlays:** `surface_container_highest` with a `backdrop-blur` of 12px–20px to create the **Glassmorphism** effect.

---

## 3. Typography: Geometric Authority
We pair the geometric precision of **Space Grotesk** with the utilitarian clarity of **Inter** to balance "The Mission" (Identity) with "The Data" (Function).

*   **Headings & Labels (Space Grotesk):** Use for all `display`, `headline`, and `label` roles. The wide apertures and geometric forms convey a futuristic, authoritative tone. 
    *   *Director’s Tip:* Use `display-lg` for mission status or altitude markers to create an editorial "hero" feel on the dashboard.
*   **Data & Body (Inter):** Use for all `title` and `body` roles. In high-stakes simulations, legibility is non-negotiable. 
    *   *Monospacing:* For telemetry logs or coordinate data, use a tabular figures setting in Inter or fallback to a clean Mono font to ensure numbers don't "jump" during live updates.

---

## 4. Elevation & Depth: Tonal Layering
Traditional drop shadows are forbidden. We use ambient light and material thickness to convey hierarchy.

*   **The Layering Principle:** Stacking surface tokens is the primary method of elevation. A `surface_container_highest` card sitting on a `surface` background provides all the "lift" required.
*   **Ambient Shadows:** If a floating modal is required, use a shadow with a 40px–60px blur at only 6% opacity. The shadow color must be a tinted version of `on_surface` (a soft, warm grey) to mimic the way light catches the edges of dark glass.
*   **The "Ghost Border" Fallback:** For buttons or input fields that require a container, use the `outline_variant` token at **15% opacity**. It should be felt, not seen.
*   **Signature Textures:** Apply a subtle linear gradient to Primary CTAs—transitioning from `primary` (#4edea3) to `primary_container` (#10b981) at a 135-degree angle. This adds a "lithium-ion" glow that flat colors cannot replicate.

---

## 5. Components: Tactical Primitives

### Buttons
*   **Primary:** Solid `primary_container` with `on_primary_container` text. Use `rounded-md` (0.75rem). Add a soft `0 0 12px` glow using the `primary` color for the `:hover` state.
*   **Tertiary:** No background, `primary` text. Use for low-priority actions like "Cancel" or "Settings."

### Input Fields & Telemetry
*   **Style:** `surface_container_highest` background with a "Ghost Border."
*   **Focus State:** Transition border opacity to 40% and add a subtle 2px emerald bottom-accent.
*   **Forbid Dividers:** Do not use lines between list items. Use `spacing-4` (0.9rem) to separate entries or alternate between `surface_container_low` and `surface_container_lowest` for row backgrounds.

### Tactical Chips
*   **UAV Status:** Use `secondary_container` for active drones.
*   **Selection:** When a drone is selected, the chip should gain a 1px `primary` ghost border and a 5% glow.

### Simulation-Specific Components
*   **The Horizon Indicator:** Use a semi-transparent `surface_variant` overlay with `primary` stroke lines for the pitch/roll ladder.
*   **Data Logs:** Set in `label-sm` (Space Grotesk) for timestamps, but `body-sm` (Inter) for the actual log message to ensure rapid scanning.

---

## 6. Do's and Don'ts

### Do:
*   **Embrace the Dark:** Allow the `surface` color to breathe. Large areas of empty obsidian make the emerald accents feel more "critical."
*   **Use Asymmetry:** Place the main feed slightly off-center and balance it with a high-density data column on the right. This breaks the "Bootstrap" look.
*   **Animate the Glow:** Interaction states should feel like hardware powering up. Use slow (300ms) ease-in-out transitions for glows.

### Don’t:
*   **Don't use pure white:** All "white" text should actually be `on_surface` (#e5e2e1). Pure white (#FFFFFF) is too harsh and breaks the cinematic immersion.
*   **Don't use sharp corners:** Even in a military context, sharp 0px corners feel dated. Stick to the `DEFAULT` (0.5rem) to `md` (0.75rem) range for a high-end, machined feel.
*   **Don't use standard icons:** Avoid "bubbly" or filled icons. Use thin-stroke (1px or 1.5px) minimalist icons that match the `outline` token weight.