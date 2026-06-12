# Capability Governance Visual Consolidation Design

**Goal:** Make the capability governance workspace read like a production B2B control surface instead of a demo form.

**Design Consensus**
- Keep backend behavior and permission semantics unchanged.
- Make capability inventory the primary task: scope selector, capability list, selected capability details, then existing grants.
- Treat grant-chain creation as an on-demand side panel that starts from the selected capability or explicit form input.
- Reduce left navigation noise by showing explanatory text only for the active workspace on desktop, and hiding it on mobile.
- Keep business labels first. Technical keys stay inside details.

**Implementation Boundary**
- Modify only frontend components, CSS, i18n copy, changelog, and focused tests.
- Do not introduce dependencies.
- Do not change API contracts, permission package rules, or backend code.

**Acceptance**
- On the capabilities page, users see capability scope and capability inventory before grant-chain creation at medium widths.
- Grant-chain creation is launched from a button instead of permanently occupying the page.
- Navigation no longer shows every item description at once.
- Capability rows scan as business rows: name and data scope first, then target, action/risk/status, grant coverage, actions.
- New text is bilingual.
- Existing access handoff and capability approval behavior remains intact.
