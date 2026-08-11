---
layout: default
title: Buildkite as an execution substrate
description: A capability-oriented description of Buildkite execution and Kaite runtime environments.
---

# Buildkite as an execution substrate

Buildkite provides a distributed system for declaring, scheduling, and
observing executable work. Its primary vocabulary is familiar: pipelines,
builds, jobs, queues, agents, steps, and artifacts. These objects describe an
execution model that applies across heterogeneous machines and runtime
environments.

A pipeline declares work and dependency. A job is a bounded execution unit. A
queue selects a class of workers. Agent tags describe worker attributes. The
Buildkite control plane assigns jobs to eligible agents. The agent starts the
job on its local execution surface and reports status, logs, and outputs.

Continuous integration and delivery are the most common workloads for this
model. The same execution model also applies to data processing, model
training, evaluation, rendering, simulation, security analysis, hardware
validation, and operational automation.

## Execution intent

Source control is one source of execution intent. A commit or pull request can
start a pipeline. Other sources include a webhook, a scheduled event, an
operational signal, a data arrival, a security finding, a model result, or the
completion of another execution.

The source event supplies context for the work. The pipeline translates that
context into executable steps, dependencies, resource requirements, and
completion conditions. Buildkite schedules the resulting jobs against the
available worker fleet.

The execution path is:

```text
event
  -> execution intent
  -> pipeline and jobs
  -> capability and queue matching
  -> agent execution
  -> result and evidence
  -> subsequent work
```

This model separates the meaning of the work from the machine that performs
it. A job can describe its requirements through queue and agent selectors
while the fleet changes underneath those selectors.

## Agents and execution surfaces

A Buildkite agent connects the control plane to a local execution surface. The
surface includes the host operating system, processors, memory, devices,
filesystems, network boundaries, credentials, installed tools, and runtime
configuration available to the job.

Agent tags make selected properties of that surface addressable to the
scheduler. A queue provides a broad placement boundary. Tags add constraints
such as hardware, operating system, architecture, software environment, or
organizational policy.

The resulting fleet can contain multiple execution classes. Examples include
Linux hosts, Apple hosts, ARM machines, NVIDIA systems, large-memory workers,
protected network segments, physical test devices, and specialized laboratory
equipment. Each class participates in the same scheduling system while
retaining its own operational and hardware requirements.

The scheduling property is capability matching. A job declares the
capabilities required for its work. An agent advertises the capabilities
available on its execution surface. Buildkite selects an agent satisfying the
declared selectors.

## Runtime environments as capability contracts

A container image provides the user-space runtime for a job. For a specialized
workload, the image also establishes a computational capability. Its contract
can include:

- language runtimes and system libraries;
- frameworks and command-line tools;
- hardware and accelerator assumptions;
- diagnostic and observability commands;
- environment identity and version information; and
- representative checks that establish the advertised behavior.

The contract defines a boundary around a package inventory. A training
environment, an inference environment, and a software-engineering environment
have different compatibility surfaces even when they share a base operating
system or language runtime.

Capability contracts give platform teams a stable unit for composition and
testing. They also give pipeline authors a stable selector for scheduling.
Image construction, worker metadata, pipeline selectors, and validation are
four views of the same execution contract.

## Kaite

Kaite supplies capability-oriented runtime environments for self-hosted
Buildkite agents. An official image contains the Buildkite agent, a pinned
Python toolchain, workload dependencies, hardware handling, diagnostics, and
lightweight observability. The image is prepared before a job arrives.

Kaite separates hardware capability from workload capability. Hardware names
identify the execution context supported by an image. Workload capabilities
identify the class of computation prepared by the image.

The current official workload capabilities are:

| Capability | Environment contract |
| --- | --- |
| `data-science` | NumPy, pandas, scikit-learn, Jupyter, and hardware-specific PyTorch |
| `training` | Hugging Face and Lightning training tooling |
| `orchestration` | Ray execution with MLflow and Weights & Biases tooling |
| `serving` | FastAPI, Gradio, and Uvicorn application interfaces |

The current official hardware classes are CPU, Apple, NVIDIA, AMD, and Intel.
CPU and Apple images are active release targets. Accelerator-specific paths
remain available for deliberate activation when matching hosts are registered.

`slim` and `full` remain compatibility names for dependency composition. The
capability aliases expose the workload meaning of those compositions. The
published image tags and aliases are documented in the [capability contract](capabilities.md).

## Identity and validation

Each Kaite image contains a baked identity at
`/etc/kaite/identity.json`. The identity records the hardware class, variant,
observability setting, and declared workload capabilities.

The supervisor loads that identity before starting the Buildkite agent. Runtime
settings for hardware, variant, observability, and capabilities are compared
with the baked values. A mismatch terminates startup. The worker therefore
advertises the identity of the image that is running on the execution surface.

The identity produces Buildkite agent tags. A worker with CPU hardware and
the training capability advertises tags equivalent to:

```text
kaite=true
kaite.hardware=cpu
kaite.capability.training=true
```

`kaite doctor` reports the baked identity and detected hardware. `kaite smoke`
checks representative framework imports for the declared capabilities and
performs the hardware check when the image expects an accelerator. These
commands connect image construction to runtime validation.

The contract is therefore expressed across four surfaces:

```text
image identity
  -> supervisor validation
  -> Buildkite agent tags
  -> pipeline selectors
```

The smoke and doctor commands provide the validation surface for the first
three stages. A pipeline provides the workload-specific execution check.

## Selecting capability from a pipeline

A pipeline selects a capability through Buildkite agent matching.
For example, a training step can request NVIDIA hardware and the training
capability:

```yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kaite.hardware: nvidia
      kaite.capability.training: "true"
```

The queue identifies the broad worker pool. The two Kaite selectors identify
the hardware class and workload environment. The job remains independent of a
specific host, container process, or image digest. Production workflows can
pin the underlying image separately through an immutable Kaite release tag.

The same pipeline can route different stages to different execution surfaces:

```yaml
steps:
  - label: ":bar_chart: prepare data"
    command: "python prepare.py"
    agents:
      queue: ai
      kaite.hardware: cpu
      kaite.capability.data-science: "true"

  - label: ":brain: train"
    command: "python train.py"
    depends_on: "prepare data"
    agents:
      queue: ai
      kaite.hardware: nvidia
      kaite.capability.training: "true"

  - label: ":satellite: serve"
    command: "python serve.py"
    depends_on: "train"
    agents:
      queue: ai
      kaite.hardware: cpu
      kaite.capability.serving: "true"
```

The pipeline describes the computational stages and their dependencies. The
worker fleet supplies the matching environments.

## Dynamic execution graphs

Buildkite pipelines can generate additional work as an execution progresses.
An initial job can inspect an input, enumerate objects, classify a condition,
or produce data for a subsequent set of jobs. The resulting steps can carry
different queues, selectors, dependencies, and concurrency limits.

This creates a progressively materialized execution graph:

```text
initial event
  -> classification
  -> fan-out across capability classes
  -> fan-in and evaluation
  -> follow-up execution
```

For example, a model-evaluation graph can prepare data on CPU workers, run
accelerated evaluation on GPU workers, aggregate results, and publish an
artifact. The graph remains one coordinated Buildkite execution while each
stage uses its own runtime contract.

Individual jobs remain finite. They receive inputs, perform bounded work,
produce outputs and evidence, and finish with a status. Subsequent executions
carry the computation forward. Logs, artifacts, metadata, and status provide
the history between those executions.

## Reasoning, execution, and hardware

AI systems introduce software that can create execution intent during a larger
task. A reasoning process can determine that it requires compilation,
evaluation, data preparation, training, rendering, or validation. Each request
can be represented as a bounded Buildkite job with explicit capability
selectors.

The responsibilities are distinct:

| Layer | Responsibility |
| --- | --- |
| Human or initiating event | Establishes purpose, constraints, and authorization |
| Pipeline | Describes executable work and dependencies |
| Buildkite | Schedules and coordinates jobs |
| Kaite | Supplies the validated runtime and hardware contract |
| Buildkite agent | Connects the selected host to the control plane |
| Hardware | Performs the computation |
| Results and evidence | Provide inputs for decisions and subsequent work |

This separation allows a reasoning environment to request specialized work
without embedding every runtime, framework, and hardware dependency in the
reasoning environment itself. Kaite supplies those dependencies as versioned
execution environments. Buildkite supplies the scheduling and execution graph.

## Direct use and downstream derivation

Kaite supports direct execution through an official image. An organization can
run the image as a self-hosted Buildkite worker and target its advertised
hardware and workload capabilities from a pipeline.

Kaite also serves as a base layer for organizational environments:

```dockerfile
FROM ghcr.io/alexhraber/kaite:<immutable-release-tag>

COPY internal-certificates/ /usr/local/share/ca-certificates/
RUN update-ca-certificates
RUN pip install --no-cache-dir internal-model-tools
COPY platform-config/ /etc/acme/
```

The derived image inherits the common Python, ML, accelerator, Buildkite,
diagnostic, and observability substrate. The organization adds its packages,
certificates, tooling, and configuration. Changes to the dependencies that
underpin a Kaite capability require rerunning the corresponding doctor and
smoke checks and owning the resulting compatibility surface.

## The execution-substrate perspective

Buildkite exposes a general execution model through its existing primitives:
pipelines describe work, jobs provide bounded units, queues and agent tags
route those units, agents attach execution surfaces, and results enable later
steps.

Kaite gives that model a tested runtime vocabulary for AI and machine-learning
work. Its image identity connects dependency composition to hardware, runtime
validation, and Buildkite agent metadata. Its capability selectors let a
pipeline request a known execution environment. Its immutable artifacts let an
organization derive a controlled extension of that environment.

Under this model, a workload selects the required capability and hardware.
Buildkite schedules the job. Kaite supplies the runtime. The execution surface
performs the work and returns evidence to the graph.
