---
name: single-cli-skill
description: Document one synchronization command with one agent skill.
problem: Choose a discoverable documentation shape for the CLI.
decision: One root skill with installation and sync fragments.
tags:
  - stack/go
  - concern/documentation
  - concern/documentation/adr
---

# Problem

The CLI has one coherent command and shared installation conventions.

# Selected variant

[Single skill](#single-skill-selected) with attached installation and command fragments.

# Searched variants

## Single skill (selected)

- Description: one discoverable skill with focused attached files.
- Benefits: one trigger, shared setup, short actionable command reference.
- Costs: split into domain skills if unrelated commands are introduced.

## Skill group

- Description: separate skills for configuration, discovery and synchronization.
- Benefits: narrower loading for a larger command surface.
- Costs: overlapping triggers for the same sync command and repeated context.
