# Current component and shell contracts

Status: M02 implemented subset. Files live in `web/src/ui/`; test through real
application interactions, not private component internals.

| Component | Contract | Current usage |
| --- | --- | --- |
| `Button` | Native `type=button`, snippet label, click action and optional native disabled state; tokenized focus/44px target | Bounded readiness retry; never submits an unrelated form |
| `SelectField` | Visible label linked by ID, supporting hint, native select with allowlisted options, optional feedback description ID | System/light/dark preference; no custom combobox keyboard model |
| `Feedback` | Escaped text/snippet, neutral/success/warning/danger presentation with words; caller selects status/alert or polite live text | Readiness (status), storage failure (alert), save success (polite) |

Readiness lives in its own component, but preserves the public health contract,
five-second browser bound, duplicate prevention and teardown abort. Error strings
are fixed, not raw network/server messages. Color adds emphasis, not meaning.

Appearance uses a real preference control, not a demonstration form. Native
selection constrains input; stored data is separately allowlisted. Storage failure
is not invalid user input: the theme remains applied and a warning is associated
with the control. It does not falsely claim a server save. Generic text inputs,
validation rules and submit-form abstractions wait for real identity requirements.

## Shell and navigation

One header, labelled application nav, focusable main and footer. The first
keyboard link skips to main; section links focus actual headings. Overview,
Appearance and Instance status are in-page destinations, not separate product
routes or authorization grants. No workspace selector, signed-in avatar, project
link, fake count or disabled product menu suggests missing capabilities exist.

Loading, ready, unavailable and retry/error states come from the actual existing
readiness endpoint. Initial product emptiness is explicitly explained. Permission-
denied, login expiry and business-data save states will be specified with identity
and protected workflows; none is simulated by this public shell.

Dialogs, menus, tables, toasts, mobile drawers, themes for rich editors and
data-density controls remain planned. Introduce them at real workflow boundaries,
including keyboard/focus/return, error and responsive contracts—not as an unused
design-system catalogue. See [principles](design-principles.md) for those gates.
