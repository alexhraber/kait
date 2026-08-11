---
layout: default
title: Buildkite as an execution substrate
description: How power users can see Buildkite beneath the CI vocabulary, and where Kaite makes that model concrete.
---

# Buildkite as an execution substrate

Buildkite is usually approached as a continuous integration and delivery
platform. That is the right entry point for most users: repositories change,
pipelines run, tests execute, artifacts are produced, and software moves
toward production.

Power users eventually encounter a more useful interpretation, one that does
not require Buildkite to become anything other than what it already is.
Beneath the familiar vocabulary of builds, jobs, queues, agents, and pipelines
is a general distributed execution model. Something creates computational
intent. That intent carries requirements and constraints. The control plane
locates compatible capacity, performs bounded work, records the result, and
allows the result to shape what happens next.

CI is one important expression of that model. It is not the boundary of it.

## The abstraction beneath CI

The conventional CI mental model begins with source control because source
control is where software delivery usually begins. A developer pushes a
commit, a pipeline starts, commands execute, tests pass or fail, and an
artifact may be deployed. It is easy to mistake this common sequence for a
limitation of the underlying architecture.

A Git event is simply one way to create execution intent. A webhook,
operational alert, scheduled event, customer action, arriving dataset, model
output, security finding, physical sensor, or preceding execution can create
the same kind of need: something happened, and computation is required.

The pipeline is therefore more than a build recipe. It is an executable
description of work and dependency. A job is a bounded execution request. A
queue participates in routing. Agent metadata describes the execution surfaces
eligible to accept work. Dependencies establish ordering. Gates constrain
progression. Dynamic steps allow earlier computation to determine later
computation.

None of this contradicts the normal CI model. It reveals the mechanics that
make the normal CI model possible.

## Buildkite as a distributed execution bus

A useful mental model is a cloud-scale execution bus. The analogy is
architectural rather than literal. In a computer, work moves toward resources
capable of performing it. General-purpose processors, accelerators, memory
systems, and attached devices serve different needs while participating in one
coherent machine.

Buildkite applies a similar principle across a distributed fleet. A unit of
work enters the system carrying requirements, and the surrounding
configuration determines which execution surfaces are eligible to receive it.
One job may require ordinary Linux. Another may require an NVIDIA accelerator,
macOS and Xcode, ARM hardware, a protected network, unusually large memory, a
proprietary compiler, a laboratory device, or a physical test rig.

The fleet does not need to become homogeneous. Heterogeneity is part of the
value. The important property is that work can describe what it requires and
the system can locate a surface able to satisfy that requirement.

This is why the Buildkite agent deserves a more precise interpretation. The
agent is commonly described as the thing that runs the job, which is
operationally reasonable but architecturally incomplete. It is more useful to
think of the agent as an attachment point between the Buildkite control plane
and an execution surface. The machine provides processors, memory, devices,
filesystems, credentials, locality, and permissions. The agent exposes that
environment and accepts compatible work on its behalf. The job remains the
computational payload.

For a power user, this means the fleet can be designed as a collection of
capabilities rather than a collection of generic runners.

## Runtime images can carry computational meaning

Container images are often treated as packaging: convenient bundles of
dependencies that make a job repeatable. For specialized workloads, an image
can mean more. It can represent a reproducible computational capability.

A model-training environment may require CUDA, PyTorch, communication
libraries, compilers, dataset tooling, and accelerator assumptions. A rendering
environment may require Blender, FFmpeg, fonts, codecs, and graphics libraries.
An agentic software-development environment may require several language
runtimes, browsers, package managers, compilers, Git, and sandboxing. A
scientific workload may depend on numerical libraries, simulation software, or
MPI.

These are not merely different lists of packages. They are different classes
of execution.

The useful boundary is therefore not “which container did this pipeline use?”
but “what capability does this execution surface reliably provide?” The
workload can request the capability while infrastructure remains free to move
between cloud, bare metal, private datacenter, Kubernetes, or a specialized
device.

## Where Kaite enters

Kaite makes this interpretation concrete for self-hosted Buildkite agents. It
gives an execution surface a stable computational identity: hardware, runtime,
diagnostics, observability, and workload capability are prepared before the
job arrives.

Kaite is not a replacement scheduler, a new Buildkite control plane, or a
generic package resolver. Buildkite still schedules the work. Kaite supplies
the known-good environment that makes the selected work possible.

Today, Kaite's official capability contract is intentionally small:

| Capability | What the environment establishes |
| --- | --- |
| data-science | Baseline numerical, notebook, and hardware-specific PyTorch tooling |
| training | Framework-neutral Hugging Face and Lightning training stack |
| orchestration | Ray execution plus MLflow and W&B experiment tooling |
| serving | FastAPI, Gradio, and Uvicorn application interfaces |

Each image records its identity in /etc/kaite/identity.json. The supervisor
validates that runtime settings agree with that baked identity. It advertises
the capability through ordinary Buildkite tags, and kaite smoke imports
representative packages for every declared capability. The image, worker,
pipeline selector, and validation surface therefore describe one contract.

The pipeline can ask for:

~~~yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kaite.hardware: nvidia
      kaite.capability.training: "true"
~~~

The request expresses computational intent. It does not encode a particular
server, image digest, or infrastructure inventory record.

## One graph, many execution surfaces

AI and machine-learning work makes this separation visible because one logical
workflow often crosses several runtime classes. Data preparation may use
ordinary CPU capacity. Compilation or preprocessing may use another
environment. Training may require accelerators. Evaluation may fan across
hundreds of jobs. Artifact conversion, performance testing, and deployment
validation may each require a different surface.

Buildkite can coordinate this as one graph while Kaite gives each branch a
defined runtime capability. The graph remains logically coherent without
forcing every stage onto one homogeneous worker.

The same pattern applies outside AI. A capability environment can represent
mobile builds, embedded toolchains, media rendering, scientific computing,
simulation, security analysis, hardware validation, or an organization-specific
network boundary. Kaite begins with AI/ML because the dependency and hardware
problem is especially sharp there, but the architectural idea is broader: make
useful runtime capability explicit enough to participate in scheduling.

## Computation can discover computation

Many problems cannot be fully decomposed before the first execution begins.
An initial job may inspect an event and discover that ten more tasks are
required. Another may discover ten thousand. Different branches may require
different capabilities, and their results may determine work that could not
have been known when the original event arrived.

An operational alert might cause log analysis, historical replay, simulation,
configuration validation, and GPU-backed evaluation to run concurrently.
When those branches converge, the result may require another graph of work—or
may show that no action is necessary. Work produced information, and the
information determined subsequent work.

Viewed strictly as CI, this can appear unusual. Viewed as distributed
execution, it is ordinary fan-out, fan-in, dependency, gating, and dynamic
step creation. The primitives are the same; the computational payload has
changed.

Continuous operation does not require an immortal process. Each execution can
begin with identifiable inputs, perform bounded work, produce identifiable
outputs and evidence, and terminate. If more work is required, the result
becomes the basis of another execution. Perpetual behavior emerges from a
succession of finite transitions, which gives the system clearer checkpoints,
failure boundaries, and history.

## AI makes the model visible

An AI agent can inspect a result and decide that more computation is necessary.
It can decompose a problem, launch investigations, request compilation,
initiate evaluation, select specialized hardware, inspect outputs, and revise
its plan.

The clean division of responsibility is:

1. The reasoning system decides what work should happen.
2. Buildkite coordinates that work.
3. Kaite defines the runtime capability.
4. The hardware performs the computation.
5. Results return to the graph and inform what happens next.

The reasoning environment does not need to contain every possible tool. An
agent can remain comparatively bounded while requesting specialized training,
evaluation, rendering, simulation, or validation work through the execution
substrate. This is stronger than constructing one enormous image because each
environment has a declared purpose and a testable boundary.

## The deepest plane is human intent

The universal mesh of abstractions is useful only because it extends human
agency. At the deepest plane sits the human: the source of purpose, judgment,
values, context, and accountability. Pipelines, agents, images, queues, and
hardware are layers through which that intent can travel without being trapped
inside one machine or one historical category.

This does not mean the human must manually operate every execution. It means
the system should make human intent more powerful without making it less
legible. A person can describe an outcome, authorize a boundary, choose a
capability, and inspect evidence while the substrate performs the mechanical
work across a heterogeneous fleet.

The abstraction is a lens, not a replacement for judgment. It lets a human
peer through the universal mesh of infrastructure and see the stable question
underneath it: what computation is required, what capability can satisfy it,
and what evidence will show that the result is trustworthy?

Kaite matters because it lowers the distance between that human question and a
real execution surface. Instead of reconstructing Python, accelerator support,
framework compatibility, diagnostics, and agent configuration in every
pipeline, a platform can publish a known capability. The human and the
workload can stay focused on the computation that is unique to the problem.

## The practical power-user model

The most useful mental model is therefore:

| Layer | Responsibility |
| --- | --- |
| Human or initiating event | Establishes purpose and creates intent |
| Pipeline | Describes executable work and dependencies |
| Buildkite | Routes and coordinates bounded execution |
| Kaite | Supplies a validated workload and hardware capability |
| Agent | Attaches the local execution surface to the control plane |
| Hardware | Performs the physical computation |
| Evidence | Determines what can happen next |

Kaite can be used as a direct destination: launch an official image and target
its capability from a pipeline. It can also be used as a foundation: inherit
from an immutable Kaite artifact and add organizational packages,
certificates, security tooling, model libraries, or configuration.

The platform team owns the common execution floor. The organization owns its
delta. The workload requests capability rather than rebuilding the substrate.

That is the point of Kaite. It is not simply an AI base image. It is a way to
make computational capability explicit enough to participate in Buildkite's
existing execution model.

Buildkite does not need to become a different product. Kaite does not need to
become a scheduler. The pipeline expresses the work, the control plane routes
it, the image supplies the capability, and the hardware performs it.

The deeper move is to see the execution model that is already present—and then
use it deliberately.
