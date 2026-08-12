---
layout: default
title: Kait · Execution environments for Buildkite
description: Capability-oriented AI and ML execution environments for self-hosted Buildkite agents.
---

<section class="hero">
  <p class="eyebrow">Buildkite execution environments</p>
  <h1>Make the environment sophisticated so the pipeline can stay simple.</h1>
  <p class="lede">
    Kait gives self-hosted Buildkite agents a known computational identity:
    the hardware, runtime, diagnostics, and workload capabilities are prepared
    before the job arrives.
  </p>
  <div class="actions">
    <a class="button" href="{{ '/execution-substrate/' | relative_url }}">Read the execution-substrate thesis</a>
    <a class="button secondary" href="{{ '/capabilities/' | relative_url }}">Choose a capability</a>
  </div>
</section>

<div class="callout">
  <strong>The operating model</strong>
  <p>
    A pipeline expresses work. Buildkite routes that work. Kait supplies a
    reproducible capability. The workload executes where the declared hardware
    and runtime contract are true.
  </p>
</div>

<h2>Start with the model</h2>

<div class="cards">
  <a class="card" href="{{ '/execution-substrate/' | relative_url }}">
    <h3>Buildkite as an execution substrate</h3>
    <p>
      A power-user perspective on pipelines, jobs, queues, and agents as the
      primitives of distributed execution.
    </p>
  </a>
  <a class="card" href="{{ '/capabilities/' | relative_url }}">
    <h3>Capability contract</h3>
    <p>
      How Kait turns runtime composition into validated workload capabilities
      such as data science, training, orchestration, and serving.
    </p>
  </a>
  <a class="card" href="{{ '/architecture/' | relative_url }}">
    <h3>Architecture</h3>
    <p>
      The supervisor, container matrix, identity model, deployment
      paths, and release topology behind the contract.
    </p>
  </a>
</div>

<h2>Use it from Buildkite</h2>

<p>
  Select the work capability and hardware in the agent selector. The pipeline
  requests intent; the worker advertises what its image can actually provide.
</p>

~~~yaml
steps:
  - label: ":brain: train"
    command: "python train.py"
    agents:
      queue: ai
      kait.hardware: nvidia
      kait.capability.training: "true"
~~~

<p>
  The selected container already contains the common
  execution substrate. The step does not spend its first minutes rebuilding
  Python, accelerator, and diagnostic machinery.
</p>

<h2>Direct destination or known-good base</h2>

<p>
  Use an official Kait image directly, or derive an organizational image from
  an immutable Kait artifact and add only the internal delta. Platform teams
  maintain the common floor; application teams and agents bring the work that
  is unique to them.
</p>

<div class="actions">
  <a class="button secondary" href="https://github.com/alexhraber/kait">View the repository</a>
  <a class="button secondary" href="https://github.com/alexhraber/kait/blob/main/examples/pipeline.yml">See the pipeline examples</a>
</div>
