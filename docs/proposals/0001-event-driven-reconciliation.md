# Proposal: Event-/Drift-Driven Reconciliation

- **Issue:** [#8 — Implement a watcher rather than regular polling](https://github.com/alperencelik/talos-operator/issues/8)
- **Status:** Draft
- **Author:** alperencelik
- **Date:** 2026-06-18

## Summary

Today every Talos controller drives progress with `reconcile.Result{RequeueAfter: ...}`
and, in a few places, blocking `time.Sleep` loops. The operator therefore re-runs
full reconciles on a timer regardless of whether anything actually changed on the
Talos side. This proposal replaces the *external-state* polling with
**drift-triggered reconciliation** using
[`kube-external-watcher`](https://github.com/alperencelik/kube-external-watcher),
and identifies where the Talos
[Events streaming API](https://github.com/siderolabs/talos/blob/main/pkg/machinery/client/events.go)
can deliver true push events.

The goal is not to remove all timers — some requeues are legitimately time-based —
but to stop reconciling when nothing has drifted, and to react quickly when it has.

## Background

### Current behavior

The operator's polling hotspots (file:line at time of writing):

| Controller | Location | Interval | Purpose |
|---|---|---|---|
| `TalosMachine` | `talosmachine_controller.go:233,283,298,349,685,718,722` | 30s | Poll machine boot/config-applied status |
| `TalosMachine` | `:282`, `:348` | — | `// TODO: Review here to make it more event driven -- maybe implement watcher` |
| `TalosControlPlane` | `taloscontrolplane_controller.go:416` | 30s | Rolling-update budget held |
| `TalosControlPlane` | `:437` (`time.Sleep`) | 10s | Blocking wait for control plane ready |
| `TalosControlPlane` | `:604,669` (`retryInterval`) | 10s | Internal retry loops |
| `TalosWorker` | `talosworker_controller.go:204` | 30s | Rolling-update budget held |
| `TalosWorker` | `:223` (`time.Sleep`) | 10s | Blocking wait for machine boot |
| `*` (all) | DryRun branches | 5m | Avoid tight-looping in dry-run |
| `TalosEtcdBackupSchedule` | `talosetcdbackupschedule_controller.go:176` | dynamic | Cron next-run (legitimately time-based) |

Two problems follow:

1. **Wasted reconciles.** A 30s requeue on every `TalosMachine` reconciles the
   resource ~2,880 times/day even when the node is healthy and unchanged. Each
   reconcile opens a Talos gRPC client and does real network I/O.
2. **Blocking worker goroutines.** `time.Sleep` inside `Reconcile`
   (`taloscontrolplane_controller.go:437`, `talosworker_controller.go:223`) ties up a
   reconcile worker slot for the duration, reducing effective concurrency.

### What `kube-external-watcher` provides

The library implements controller-runtime's `source.Source`. It runs one goroutine
per watched resource, polls the *external* system on an interval, and enqueues a
`reconcile.Request` **only when the observed external state differs from the desired
state**. When state matches, no reconcile happens.

Key API (`package watcher`, import `github.com/alperencelik/kube-external-watcher/watcher`):

```go
// You implement this against the Talos API.
type ResourceStateFetcher interface {
    GetDesiredState(ctx context.Context, key types.NamespacedName) (any, error)
    FetchExternalResource(ctx context.Context, objKey any) (any, error)
    TransformExternalState(raw any) (any, error)
    IsResourceReadyToWatch(ctx context.Context, key types.NamespacedName) bool
}

w := watcher.NewExternalWatcher(fetcher,
    watcher.WithDefaultPollInterval(30*time.Second),
    watcher.WithLogger(log),
    watcher.WithAutoRegister(mgr.GetCache(), &talosv1alpha1.TalosMachine{},
        func(obj client.Object) watcher.ResourceConfig {
            m := obj.(*talosv1alpha1.TalosMachine)
            return watcher.ResourceConfig{ResourceKey: m.Spec.Endpoint /*, PollInterval*/}
        },
        watcher.AutoRegisterWithFilter(watcher.EventFilter{ /* gen change, etc. */ }),
    ),
)

ctrl.NewControllerManagedBy(mgr).
    For(&talosv1alpha1.TalosMachine{}).
    WatchesRawSource(w). // replaces RequeueAfter polling
    Complete(r)
```

`StateComparator` is optional and defaults to deep-equality
(`NewDeepEqualComparator()`). The watcher also exposes `Register`/`Unregister`/
`IsRegistered`/`LastDrift` for manual control and surfaces Prometheus metrics.

> **Honest framing:** this is *drift-triggered polling*, not zero-poll. The library
> still calls the Talos API on an interval per node — but it centralizes that polling
> in dedicated, rate-limited goroutines and stops the wasteful *reconciles*. That
> directly satisfies the performance motivation in #8.

### True event source: the Talos Events API

For one class of signal we can do better than polling. The Talos machinery client
exposes a streaming events API (confirmed in `pkg/machinery/client/events.go@v1.13.0`):

```go
func (c *Client) Events(ctx, opts...) (MachineService_EventsClient, error)
func (c *Client) EventsWatch(ctx, func(<-chan Event), opts...) error
func (c *Client) EventsWatchV2(ctx, chan<- EventResult, opts...) error
```

These deliver sequence/phase events (boot progress, config-apply, upgrade
lifecycle, etc.) as a server push. This is the natural fit for the
"is the machine booted / has config applied" polling in `TalosMachine`, which is
exactly what the `:282`/`:348` TODOs call out.

## Proposal

A phased migration. Each phase is independently shippable and measurable.

### Phase 1 — `TalosMachine` drift watcher (highest ROI)

Introduce `internal/watcher` with a `TalosMachineFetcher` implementing
`ResourceStateFetcher`:

- `GetDesiredState`: read the `TalosMachine` CR — desired Talos `version`, and the
  config hash derived from `status.config` / `bundleConfig`.
- `FetchExternalResource` / `TransformExternalState`: open a Talos client to
  `spec.endpoint` and read observed version (`tc.Version`) and service/boot status
  (`tc.GetServiceStatus`), normalized into the same shape as desired state.
- `IsResourceReadyToWatch`: return true once the machine has an endpoint and a config
  has been applied (so we don't watch machines that aren't provisioned yet).

Wire it with `WatchesRawSource` and `WithAutoRegister` keyed on `spec.endpoint`.
Remove the 30s `RequeueAfter` status-polling at
`talosmachine_controller.go:233,283,298,349`. Keep short error requeues.

**Outcome:** steady-state healthy machines stop reconciling entirely; an
out-of-band version/config change is reflected within one poll interval.

### Phase 2 — Boot/sequence status via Talos Events API

For the boot-and-config-apply waits (the `:282`/`:348` TODOs and the
`time.Sleep` loops in CP/Worker), run a manager `Runnable` that holds a long-lived
`EventsWatchV2` stream per machine and enqueues a reconcile when a relevant
sequence/phase event arrives. This removes the blocking `time.Sleep` at
`taloscontrolplane_controller.go:437` and `talosworker_controller.go:223`, freeing
reconcile workers.

> Phase 2 is independent of the library; it's a direct use of the Talos client.
> It can land before or after Phase 1.

### Phase 3 — Control plane / worker version & etcd drift

Extend the drift-watcher pattern to `TalosControlPlane` and `TalosWorker`:
observed K8s version (`observedKubeVersion`) and etcd member health. This replaces
the readiness re-polling and lets rolling updates progress on actual readiness
signals rather than fixed 30s budget checks.

### Explicitly out of scope (keep time-based)

- `TalosEtcdBackupSchedule` dynamic requeue (`:176`) — cron scheduling is correct.
- DryRun 5-minute requeues — these are loop-dampeners, not external state; leave or
  convert to no-requeue depending on desired UX.
- Short error-retry requeues (`Requeue: true`) — keep for backoff.

## Design details

```
internal/
  watcher/
    fetcher_machine.go      // TalosMachineFetcher (ResourceStateFetcher)
    fetcher_controlplane.go // Phase 3
    state.go                // normalized state structs + comparator
    events.go               // Phase 2: Talos EventsWatchV2 Runnable
```

- **Client reuse:** fetchers should reuse the existing `pkg/talos` client
  constructor (`talos.NewClient`) and the credentials already resolved from
  `bundleConfig` so auth logic stays in one place.
- **Comparator:** start with the default deep-equality comparator over a small
  normalized struct (`{Version, ConfigHash, Ready}`) to avoid noisy drift from
  fields we don't manage.
- **Registration lifecycle:** `WithAutoRegister` ties watch start/stop to CR
  create/delete via the manager cache; on CR delete the per-resource goroutine is
  cleaned up by `Unregister`.
- **Failure handling:** a fetch error should not be treated as drift; it should be
  logged/counted and retried on the next interval (confirm library behavior, else
  guard in `TransformExternalState`).

## Risks & considerations

- **New dependency** (`kube-external-watcher`) and added goroutines (one per
  machine). Bounded by node count; the library rate-limits the workqueue. Already
  proven in `kubemox`.
- **Drift definition is subtle.** Over-broad state structs cause reconcile storms;
  under-broad ones miss real drift. Keep the normalized state minimal and expand
  deliberately.
- **Two mechanisms coexist.** During migration, drift watcher + residual requeues
  run together. Land per-controller behind the existing reconcile logic so behavior
  is additive, then remove the timers.
- **Auth/endpoint churn.** Endpoints/credentials can change; the fetcher must read
  them fresh from the CR/bundle each cycle rather than caching at registration.

## Migration plan

1. Add dependency; scaffold `internal/watcher` + `TalosMachineFetcher`.
2. Wire `WatchesRawSource` into `TalosMachine` **alongside** existing requeues; ship
   and observe drift/reconcile metrics.
3. Remove the 30s status-poll requeues from `TalosMachine`.
4. Phase 2: Events-API Runnable; delete the `time.Sleep` waits.
5. Phase 3: extend to control plane / worker.

## Success metrics

- Reconciles/min for `TalosMachine` drops to ~0 in steady state (today: one per
  machine per 30s).
- Out-of-band version/config change detected within one poll interval.
- No blocking `time.Sleep` left in `Reconcile` paths.
- No reconcile-worker starvation under N nodes.

## Implementation status

**Phase 1 scaffolded** (additive — existing `RequeueAfter` polling left in place):

- `internal/watcher/machine_fetcher.go` — `TalosMachineFetcher` implementing
  `ResourceStateFetcher`; compares desired `spec.version` against the node's
  reported Talos version. `IsResourceReadyToWatch` gates on `Status.State ==
  Available`. Reuses the controller's `GetBundleConfig` via a `BundleConfigResolver`
  interface (no duplication).
- `internal/controller/talosmachine_controller.go` — optional `ExternalWatcher
  source.Source` field, wired via `.WatchesRawSource` when set.
- `cmd/main.go` — constructs the `ExternalWatcher` directly (auto-register off
  `mgr.GetCache()`, 30s default poll, metrics under `talosmachine`) and attaches
  it to the reconciler.
- Dependency: `github.com/alperencelik/kube-external-watcher v0.1.0`.

Next: observe drift/reconcile metrics in a cluster, then remove the 30s status-poll
requeues at `talosmachine_controller.go:233,283,298,349`.

## Open questions

- ~~Does the library treat a fetch error as "no drift"?~~ **Resolved:** a fetch error
  is logged + counted and the poll returns without enqueuing — no reconcile-on-error
  storms (`resource_watcher.go:poll`). A briefly-unreachable node is therefore safe.
- Per-CR configurable poll interval — expose a `spec.pollInterval` field
  (`ResourceConfig.PollInterval` supports it), or keep the operator-wide default via
  `WithDefaultPollInterval`?
- Should the fetcher also implement the optional `ResourceStatusUpdater` to sync
  observed node state into `.status` on every poll? (Deferred — risks status-write
  races with the controller; revisit if needed.)
- Phase 2 vs Phase 1 ordering — is the Events API mature/stable enough across the
  supported Talos versions to lead with it?
