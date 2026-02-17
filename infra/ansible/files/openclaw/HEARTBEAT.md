# HEARTBEAT.md

## Periodic Checks (rotate through, 2-4x daily)
- [ ] Review recent memory/YYYY-MM-DD.md files — distill insights into MEMORY.md if needed
- [ ] Check for stale info in MEMORY.md that should be cleaned up

## Track State
Use `memory/heartbeat-state.json` to track last check times. Don't repeat checks within 2 hours.

## Rules
- Late night (23:00-08:00): HEARTBEAT_OK unless urgent
- Nothing new since last check: HEARTBEAT_OK
- If you do proactive work (organize files, update memory), mention it briefly
