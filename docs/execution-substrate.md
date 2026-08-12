---
layout: default
title: Buildkite as an execution substrate
description: A capability-oriented description of Buildkite execution and Kait runtime environments.
---

# Buildkite as an execution substrate

Buildkite provides a distributed execution substrate for declaring,
scheduling, routing, and observing computational work. Its familiar
primitives—pipelines, builds, jobs, queues, agents, steps, artifacts,
dependencies, and gates—form a general execution model across heterogeneous
machines and runtime environments.

Continuous integration and delivery are first-class applications of that
model. The same primitives also coordinate model training, evaluation, data
processing, rendering, simulation, security analysis, hardware validation,
operational automation, and other bounded computation. The workload changes;
the execution model remains.

A pipeline describes executable work and dependency. A job is a bounded
execution unit. Queues and agent tags form the placement surface. Agents expose
execution capacity to the Buildkite control plane. Buildkite matches work to
eligible agents, coordinates its progression, and records the resulting
status, logs, artifacts, and metadata.

For power users, this is the productive abstraction: computational intent
enters Buildkite, is resolved against available execution capability, runs on
the appropriate hardware, and produces results that can drive subsequent
computation.

## Execution intent

Source control is one producer of execution intent. Commits and pull requests
create work, as do webhooks, schedules, operational alerts, data arrivals,
security findings, model outputs, external events, and preceding executions.

The initiating event provides context. The pipeline translates that context
into executable steps, dependencies, capability requirements, gates, and
completion conditions. Buildkite schedules the resulting jobs across the
available execution fleet.

The execution path is:

```text
event
  -> execution intent
  -> pipeline and jobs
  -> capability matching
  -> execution surface
  -> result and evidence
  -> subsequent work
```

This separation between computational intent and physical execution is one of
Buildkite's core properties. Jobs declare what they require through queues and
agent selectors while the infrastructure satisfying those selectors evolves
independently.

## Agents as execution surfaces

A Buildkite agent attaches local compute to the control plane. Behind that
attachment point is an execution surface composed of operating system,
processors, memory, accelerators, devices, filesystems, network locality,
credentials, installed software, and runtime configuration.

Queues establish broad execution boundaries. Agent tags expose selected
properties of those surfaces to scheduling. Together they organize a fleet
around executable capability.

A Buildkite fleet can span ordinary Linux hosts, Apple systems, ARM machines,
NVIDIA GPU workers, high-memory hosts, protected network segments, physical
test devices, laboratory equipment, and specialized internal infrastructure
while remaining accessible through the same execution plane.

For a power user, the useful scheduling abstraction is capability matching. A
job declares the properties required for its work, an agent advertises
properties of its execution surface, and Buildkite matches jobs against
eligible agents through queues and agent tags.

## Runtime environments as capability contracts

Runtime images make execution capabilities reproducible.

For specialized workloads, an image defines the user-space contract under
which a class of computation executes: language runtimes, frameworks,
command-line tools, accelerator support, system libraries, diagnostics,
observability hooks, environment identity, and representative validation.

A training environment, an inference environment, a rendering environment,
and an agentic software-engineering environment are different computational
contracts even when they share the same operating system or programming
language.

This gives the execution fleet semantic structure. Infrastructure can
advertise that a validated environment exists for a known class of work,
alongside the hardware and operating-system properties of the host.

## Kait

The execution-substrate model gains a concrete runtime boundary when the
properties exposed through agent tags correspond to reproducible execution
environments. Kait provides that layer for AI and machine-learning workloads.

Kait supplies capability-oriented runtime environments for self-hosted
Buildkite agents. An official container contains the
Buildkite agent integration, a pinned Python toolchain, workload dependencies,
hardware handling, diagnostics, and lightweight observability. The execution
surface is prepared and validated before a job arrives, allowing the worker to
advertise a known computational capability alongside the characteristics of
the host on which it runs.

Kait separates hardware capability from workload capability. Hardware
identifies the execution context available to the runtime. Workload capability
identifies the class of computation the environment is prepared to perform.

The current official workload capabilities are:

| Capability | Environment contract |
| --- | --- |
| `data-science` | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch |
| `training` | Hugging Face and Lightning training tooling |
| `orchestration` | Ray execution with MLflow and Weights & Biases tooling |
| `serving` | FastAPI, Gradio, and Uvicorn application interfaces |

The current hardware classes are CPU, Apple, NVIDIA, AMD, and Intel. Apple GPU
execution is modeled as a native macOS arm64 surface: Apple Container can run
Linux OCI images, but it does not expose Metal to those Linux VMs. The Apple
contract is currently disabled like the other accelerator classes, so Kait
does not build or release Apple profiles. Apple Silicon users run the active
multi-architecture Linux CPU profiles and advertise `kait.hardware=cpu`.

Kait publishes six explicit profiles across the hardware matrix: the
compatibility profiles `slim` and `full`, plus the workload profiles
`data-science`, `training`, `orchestration`, and `serving`. The workload
profiles are real environment compositions, not aliases to a larger image.
Linux profiles are OCI images and the native Apple path is reserved. `training`
composes the data-science foundation; `orchestration` and `serving` remain
independent so their selectors describe meaningful execution environments;
`full` composes all four official workload capabilities.

These two dimensions give Buildkite's scheduling surface computational
meaning. A worker advertises the hardware and runtime capability available at
that execution surface.

For example:

```text
kait=true
kait.hardware=nvidia
kait.capability.training=true
```

A pipeline can address that capability directly:

```yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kait.hardware: nvidia
      kait.capability.training: "true"
```

The request identifies the execution requirement: NVIDIA-backed Kait training
capability. Buildkite routes the job onto an eligible execution surface. Kait
supplies the training runtime. The pipeline expresses the computational
capability required for the job.

The container's `/etc/kait/identity.json` is produced during construction by
the same contract resolver that selects its manifests. The reserved native
path uses the same resolver when it is re-enabled.
`kait doctor` reports that identity and host hardware evidence, while
`kait smoke` executes bounded local checks for every advertised capability.
Runtime environment variables may assert the baked values but cannot create a
capability claim or replace a different execution identity. Before Apple is
re-enabled, its data-science smoke path must prove
`torch.backends.mps.is_available()` and execute a small tensor operation on
the MPS device.

This is the operating model Kait makes concrete:

```text
Buildkite       -> execution substrate
queues and tags -> capability routing
Kait           -> runtime contract
agent           -> attachment to the execution surface
hardware        -> computation
pipeline        -> composition of executions
```

## Heterogeneous execution graphs

Once runtime capability is explicit, a single pipeline can use different
classes of compute naturally.

Data preparation can run on CPU workers. Training can move to NVIDIA workers.
Evaluation can fan out across another fleet. Serving validation can return to
CPU infrastructure. Other branches can target protected networks, different
architectures, or specialized runtime profiles.

Each stage declares what it needs. Buildkite resolves those requirements
independently.

The execution graph therefore assigns runtime placement to individual jobs.
The pipeline remains the logical composition of the computation, while each
job selects the execution surface that satisfies its requirements.

For example:

```yaml
steps:
  - label: ":bar_chart: prepare data"
    key: "prepare-data"
    command: "python prepare.py"
    agents:
      queue: ai
      kait.hardware: cpu
      kait.capability.data-science: "true"

  - label: ":brain: train"
    key: "train"
    command: "python train.py"
    depends_on: "prepare-data"
    agents:
      queue: ai
      kait.hardware: nvidia
      kait.capability.training: "true"

  - label: ":satellite: serve"
    command: "python serve.py"
    depends_on: "train"
    agents:
      queue: ai
      kait.hardware: cpu
      kait.capability.serving: "true"
```

Buildkite also supports dynamic pipelines. Earlier computation can inspect
inputs or results and materialize additional work at runtime. A classification
step can discover that hundreds of independent evaluations are required. Those
jobs can fan across the appropriate capability class, converge into
aggregation, and produce another stage of computation.

The graph is progressively constructed from what execution discovers.

Individual jobs remain finite. They receive inputs, perform bounded work,
produce outputs and evidence, and finish with a status. Subsequent executions
carry the computation forward. Logs, artifacts, metadata, and status provide
the history between those executions.

## Reasoning, execution, and hardware

AI systems introduce software that can create execution intent during a larger
task. A reasoning process can determine that it requires compilation,
evaluation, data preparation, training, rendering, simulation, or validation.
Each request can be represented as a bounded Buildkite job with explicit
capability selectors.

The responsibilities are distinct:

| Layer | Responsibility |
| --- | --- |
| Human or initiating event | Establishes purpose, constraints, and authorization |
| Pipeline | Describes executable work and dependencies |
| Buildkite | Schedules and coordinates jobs |
| Kait | Supplies the validated runtime and hardware contract |
| Buildkite agent | Connects the selected host to the control plane |
| Hardware | Performs the computation |
| Results and evidence | Provide inputs for decisions and subsequent work |

The reasoning environment can remain bounded while specialized work is
requested through Buildkite. Kait supplies the requested dependencies as
versioned execution environments. Buildkite supplies the scheduling and
execution graph.

This creates a distributed computer composed of intentionally specialized
execution surfaces with explicit provisioning boundaries.

## Direct use and downstream derivation

Kait supports direct execution through an official Linux container. An
organization can run a Linux image as a self-hosted Buildkite worker, including
on Apple Silicon through Docker or Apple Container, then target the advertised
CPU hardware and workload capabilities from a pipeline. The native Apple
bundle remains reserved while Apple is inactive.

Kait also serves as a base layer for organizational container environments:

```dockerfile
FROM ghcr.io/alexhraber/kait:<immutable-release-tag>

COPY internal-certificates/ /usr/local/share/ca-certificates/
RUN update-ca-certificates
RUN pip install --no-cache-dir internal-model-tools
COPY platform-config/ /etc/acme/
```

The derived image inherits the common Python, ML, accelerator, Buildkite,
diagnostic, and observability substrate. The organization adds its packages,
certificates, tooling, and configuration. Changes to dependencies that
underpin a Kait capability require rerunning the corresponding doctor and
smoke checks and owning the resulting compatibility surface.

## The power-user model

The productive way to use Buildkite is to separate computational intent from
execution placement.

Pipelines express work.

Jobs bound execution.

Dependencies describe computational relationships.

Queues and agent tags route work according to capability.

Agents attach machines and environments to the execution plane.

Dynamic pipelines allow computation to produce further computation.

Gates control progression.

Artifacts, logs, metadata, and status preserve the result.

Kait extends this model by giving AI and machine-learning execution surfaces
stable runtime identities that pipelines can address directly. A workload
declares the capability and hardware it requires; Buildkite resolves that
request against matching execution capacity; Kait supplies the validated
runtime contract; the selected hardware performs the computation; and the
result returns to the execution graph as input to whatever happens next.

This is the power-user model for the tool in front of us: Buildkite as a
distributed execution substrate whose existing primitives route computational
intent across heterogeneous, capability-defined infrastructure. Pipelines
compose the work, jobs bound its execution, queues and agent tags determine
placement, agents attach execution surfaces to the control plane, and Kait
gives specialized AI and machine-learning environments a reproducible identity
within that fabric. The workload changes; the execution model remains.
