---
name: container-management:pod-diagnosis
description: >
  Use when a user asks to diagnose, inspect, troubleshoot, or find the root
  cause of a specific Pod failure, non-running Pod, or pod-level issue in a
  Kubernetes cluster managed by the DCE/kpanda module. Also use for Chinese
  requests like 排查 Pod 故障原因、某个 pod 异常、pod 启动失败、pod 无法运行、
  pod 一直重启、CrashLoopBackOff、OOMKilled、ImagePullBackOff、Pending、Evicted、
  or Terminating. Use `container-management:cluster-diagnosis` instead for a
  cluster-wide symptom or when no individual Pod has been identified yet.
---

# Kpanda Pod Diagnosis

Diagnose unhealthy or failing pods through a standardized inspection workflow.

**REQUIRED SUB-SKILL:** Use `dce` for all command execution, auth checks, and catalog discovery.

## Collection Flow

### Step 1 — Resolve One Exact Target and Capture Its Snapshot
- This skill diagnoses one exact `cluster` / `namespace` / `pod` target. When
  all three are known, call the following command once; do not first scan the
  entire cluster:

  `dce container-management core get-pod --cluster <cluster> --namespace <namespace> --name <pod> -o json`

- If only the namespace is missing, resolve it with
  `dce container-management core list-cluster-pods --cluster <cluster> --name <pod> --all -o json`, require an exact-name match, and ask the user to choose when more than one namespace matches.
- If the Pod name is missing, present a bounded candidate list and require the
  user to select one before diagnosis. `Pending`, `Failed`, `Unknown`, or
  non-ready/restarting container state are candidates; `Succeeded` is terminal
  rather than `Completed`, and CrashLoopBackOff can still have `Running` phase.
- If the target does not exist, report that result and stop. Do not silently
  substitute another Pod.

### Step 2 — Collect Targeted Pod Events
- `dce container-management core list-events --cluster <cluster> --namespace <namespace> --kind Pod --kind-name <pod> --all -o json`
- The namespaced query is the default evidence source: `--kind-name` filters
  the involved Pod. Do not use `--name` here; it fuzzy-matches the Event name.
- Only if the namespaced endpoint cannot provide the needed evidence, use the
  cluster-wide fallback
  `dce container-management core list-cluster-events --cluster <cluster> --kind Pod --name <pod> --all -o json`
  and retain only results in `<namespace>`. Do not run both event queries by
  default.

### Step 3 — Retrieve Bounded Logs from Relevant Containers
- Select only failing, recently restarted, or non-ready containers from the
  Step 1 snapshot.
- `dce container-management insight get-pod-container-log --cluster <cluster> --namespace <namespace> --name <pod> --container <container> --page-size <bounded-page-size> -o json`
- Use an explicit recent time window when available and inspect only the needed
  pages. Do not dump all historical logs.
- Check for stack traces, OOM signals, exit codes, or missing dependencies.

### Step 4 — Inspect Related Workloads
If the pod is owned by a controller:
- `dce container-management core list-pods --cluster <cluster> --namespace <namespace> --kind <owner-kind> --kind-name <owner-name> --all -o json`
- Use only owner kinds supported by this command's catalog. This returns
  related Pods and can compare restart/readiness patterns; it cannot by itself
  prove a controller's desired replica count or selector mismatch.

### Step 5 — Node Affinity and Resource Analysis
- Only for a scheduled Pod with `<node>`:
  - `dce container-management core get-node --cluster <cluster> --name <node> -o json`
  - `dce container-management core list-pods-by-node --cluster <cluster> --node <node> --all -o json`
- Compare the Pod's tolerations, node selectors, affinity rules, and resource
  requests/limits with the returned node evidence. A Pending Pod without a
  node cannot establish node pressure or affinity failure from these calls
  alone; state the limit and rely on Pod events/spec evidence.

## User omitted cluster name
Run `dce container-management cluster list-clusters --all -o json`, present a
bounded choice list, and ask the user to pick one.

## User omitted pod name
Run `dce container-management core list-cluster-pods --cluster <cluster> --all -o json`, present a bounded candidate list using composite status and container state rather than phase alone, and ask the user to pick one.

## Auth not established
Stop and instruct user to run `dce auth login --hostname <host>`.

## Output Format

Present the final answer as structured Markdown. Do not include a step-by-step
tool execution log, skill loading details, API retry details, JSON parsing
details, or other internal process unless the user explicitly asks for them.
If data is incomplete, explicitly say that the judgment is based on currently
available data in the conclusion.

Use these top-level sections in this order. Treat the template as the report
spine, not as a limit on evidence: preserve domain-specific tables and details
inside the matching sections when they are needed to support the conclusion.

# Conclusion

Use 1-2 sentences to state the current judgment, risk level
(`normal` / `watch` / `risk` / `critical`), and the most important issue.
For user-facing answers, localize the section title and risk labels to the
user's language.

## Key Metrics

Start with a Markdown summary table with 3-6 key indicators. Prefer these
fields when available: Pod phase, restart count, node, container state, last
exit code, warning event count, and latest error reason.

| Metric | Current Value | Status |
|--------|---------------|--------|
| Pod phase | `<value>` | `<normal/watch/risk/critical>` |

If the Pod has meaningful container, event, or log evidence, include supporting
detail tables under this section, such as:

- Pod overview: `Namespace | Pod | Phase | Node | Restarts | Age`
- Container states: `Container | Ready | Restart Count | State | Last State | Exit Code`
- Events: `Type | Reason | Message | Last Seen`
- Log evidence: short excerpts only, grouped by container, without dumping full logs

## Main Findings

Use a numbered list only for independent findings. Each finding must explain
the impact. Do not invent findings to reach a target count. Do not omit the
decisive evidence: include the event reason, container state, exit code, or
log signal that supports each finding.

## Cause Analysis

Analyze 2-3 causes around the main findings. For each cause, include:

Cause N: `<cause>`

Evidence: `<specific event, container state, exit code, or log excerpt>`.

Impact: `<user-visible or operational impact>`.

## Recommended Actions

Group concrete actions by:

### Immediate

### Monitor

### Optimize Later

## Follow-up Questions

Provide 2-3 copyable follow-up questions in the user's language. They should
guide the user toward detailed events/logs, remediation planning, or an
exportable stakeholder report.

## Rules

- Prefer `-o json` for machine-readable output.
- Do not guess flags or body shape. Confirm with `dce commands show` before executing unfamiliar commands.
- Report empty API responses as "no resources found" rather than silently skipping.
- Do not perform remediation (restart, delete, scale). This skill is read-only.
- If multiple pods are affected, prioritize by restart count and age — most started/recent failures first.
- Put the conclusion first. Do not write the final answer as a troubleshooting
  transcript.
- Use tables for indicators whenever possible.
- Recommended actions must be specific and executable.
