# Docker Compose Review Checklist

- [ ] default services are frontend, gateway, orchestrator, code-agent, web-agent.
- [ ] remote Agents remain dynamically registerable.
- [ ] Frontend reaches Gateway only.
- [ ] internal ports are not unnecessarily public.
- [ ] SQLite uses a persistent volume and documented WAL behavior.
- [ ] MySQL/Redis are not mandatory defaults.
- [ ] health checks represent readiness.
- [ ] Mock startup needs no API key.
- [ ] real-model missing secret fails clearly.
- [ ] Compose and `.env.example` contain no secret.
- [ ] no secret is placed in `VITE_*`.
- [ ] shutdown/cancellation is handled.
- [ ] volume restart and Mock smoke pass.
- [ ] optional observability profile does not block core startup.
- [ ] documentation matches actual service names, ports, profiles, and variables.
