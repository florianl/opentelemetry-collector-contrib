# Issue management

To help provide a consistent process for seeing issues through to completion, this section details some guidelines and
definitions to keep in mind when triaging issues.

### Roles

Determining the root cause of issues is a shared responsibility between those with triager permissions, code owners,
OpenTelemetry community members, issue authors, and anyone else who would like to contribute.

#### Triagers

Contributors with [triager](/#Contributing) permissions can help move
issues along by adding missing component labels, which help organize issues and trigger automations to notify code owners. They can
also use their familiarity with the Collector and its components to investigate issues themselves. Alternatively, they may point issue
authors to another resource or someone else who may know more.

#### Code Owners

In many cases, the code owners for an issue are the best resource to help determine the root cause of a bug or whether an enhancement
is fit to be added to a component. Code owners will be notified by repository automations when:

- a component label is added to an issue
- an issue is opened
- the issue becomes stale

Code owners may not have triager permissions on the repository,
so they can help triage through investigation and by participating in discussions. They can also help organize issues by
[adding labels via comments](/CONTRIBUTING.md#adding-labels-via-comments).

#### Community Members

Community members or interested parties are welcome to help triage issues by investigating the root cause of bugs, adding input for
features they would like to see, or participating in design discussions.

### Triage process

Triaging an issue requires getting the issue into a state where there is enough information available on the issue or understanding
between the involved parties to allow work to begin or for the issue to be closed. Facilitating this may involve, but is not limited to:

- Determining whether the issue is related to the code or documentation, or whether the issue can be resolved without any changes.
- Ensuring that a bug can be reproduced, and if possible, the behavior can be traced back to the offending code or documentation.
- Determining whether a feature request belongs in a component, should be accomplished through other means, or isn't appropriate for a component at this time.
- Guiding any interested parties to another person or resource that may be more knowledgeable about an issue.
- Suggesting an issue for discussion at a SIG meeting if a synchronous discussion would be more productive.

It is recommended that a triager should follow these steps when a new issue is reported:

1. Assess the validity of the issue. Here's a rough outline on how to assess:
   - Does the issue include a clear description of the problem?
      - Whether it’s a bug report, an enhancement proposal, or a request for a new component, the purpose of the issue should be easy to understand.
   - The following outline is applicable to bugs or enhancements:
      - Does the issue specify the relevant version or environment details?
      - Does it provide the user’s configuration, the expected behavior, and any other relevant context?
      - If it's reported as a bug, are the steps to reproduce clearly outlined?
      - For enhancements, does the user clearly outline the expected behavior and provide relevant examples or use cases?
   - For new component requests:
      - Does the user explain the purpose and intended use of the component?
      - Has the user provided example configurations, interfaces, or scenarios where the component would be applied?
      - Will the user contribute the necessary code, or they are expecting someone from upstream to volunteer?
      - Will the user be available to provide feedback, clarify requirements, or help with testing and maintenance?
   - If the issues lacks sufficient information, ask the author clarifying questions:
      - Collector configuration (missing or incomplete configurations are not uncommon)
      - The expected vs. actual behavior
      - Clear and complete steps to reproduce the issue
      - Collector version
      - Any other questions that might fit

2. If the issue is deemed valid and requires attention, apply appropriate labels based on its nature:
   - For bugs, add the `bug` label along with `waiting-for-codeowners`.
   - For enhancements, use the `enhancement` label and also add `waiting-for-codeowners`.
   - If the issue is about a new component, add the `sponsor-needed` label. We should encourage the user to join our SIG meeting and propose the new component to a broader audience. They can also reach out in the #otel-collector-dev channel on CNCF Slack.
   - Check out [state diagram](#state-diagram) for detailed information about all labels.

3. An issue may be considered invalid for the following reasons:
   - The user is using an incorrect configuration.
       - In such cases, a brief comment and a link to documentation should be sufficient to close the issue.
   - The issue was filed in the wrong repository:
       - For example, we might receive an issue for `otlpexporter` in contrib. It is best to reach out in the #otel-collector-dev channel on CNCF Slack and ask project maintainers to transfer the issue.

4. An issue may be considered complete for following reasons:
   - The enhancement/bugfix PR has been merged.
   - The enhancement/bugfix has already been addressed in the newer version.
   - The new component has been added to the repository.

#### Triage process flowchart 

```mermaid
flowchart TD
    Start{Check validity}

    Start -->|Unclear| WaitingAuthor[Tag:<br><code>waiting-for-author</code><br>Ask for config, repro steps,<br>expected vs actual result]
    Start -->|Invalid| Transfer[Close or<br>transfer issue:<br>wrong repo, bad config, etc.]
    Start -->|Valid| Classify

    Classify -->|Bug| EnrichBug[Tag:<br><code>bug</code>,<br><code>waiting-for-codeowners</code>,<br>Optional: <code>bug:perf/crash</code>,<br><code>workaround:yes/no</code>,<br><code>release:blocker</code>]

    Classify -->|Enhancement| EnhancementLabel[Tag:<br><code>enhancement</code>,<br><code>waiting-for-codeowners</code>]

    Classify -->|Documentation| DocLabel[Tag:<br><code>documentation</code>,<br><code>waiting-for-codeowners</code>]

    Classify -->|New Component| NewComponent[Tag:<br><code>sponsor-needed</code>]
    NewComponent --> Expectations[Set expectations:<br>Will user contribute code?<br>Will they provide feedback/support?<br><br>Suggest joining SIG,<br>propose via SIG,<br>or reach out on<br><code>#otel-collector-dev</code> Slack]
```

#### Issue assignment

Issues are assigned for someone to work on by a triager when someone volunteers to work on an issue. Assignment is intended to prevent duplicate work by making it visible who is
working on a particular task. A person who is assigned to the issue may be assigned to help triage the issue and implement it, or can be assigned after the issue has already been
triaged and is ready for work. If someone who is assigned to an issue is no longer able to work on it, they may request to be unassigned from the issue.

### Label Definitions

| Label                | When to apply                                                                                                                                                                                                  |
| -------------------- |----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `bug`                | Something that is advertised or intended to work isn't working as expected.                                                                                                                                    |
| `enhancement`        | Something that isn't an advertised feature that would be useful to users or maintainers.                                                                                                                       |
| `flaky test`         | A test unexpectedly failed during CI, showing that there is a problem with the tests or test setup that is causing the tests to intermittently fail.                                                           |
| `documentation`      | This is a collector usability issue that could likely be resolved by providing relevant documentation. Please consider adding new or improving existing documentation before closing issues with this label.   |
| `good first issue`   | Implementing this issue would not require specialized or in-depth knowledge about the component and is ideal for a new or first-time contributor to take.                                                      |
| `help wanted`        | The code owners for this component do not expect to have time to work on it soon, and would welcome help from contributors.                                                                                    |
| `discussion needed`  | This issue needs more input from the maintainers or community before work can be started.                                                                                                                      |
| `needs triage`       | This label is added automatically, and can be removed when a triager or code owner deems that an issue is either ready for work or should not need any work. See also the [triaging process](#triage-process). |
| `waiting for author` | Can be applied when input is required from the author before the issue can move any further.                                                                                                                   |
| `priority:p0`        | A critical security vulnerability or Collector panic using a default or common configuration unrelated to a specific component.                                                                                |
| `priority:p1`        | An urgent issue that should be worked on quickly, before most other issues.                                                                                                                                    |
| `priority:p2`        | A standard bug or enhancement.                                                                                                                                                                                 |
| `priority:p3`        | A technical improvement, lower priority bug, or other minor issue. Generally something that is considered a "nice to have."                                                                                    |
| `release:blocker`    | This issue must be resolved before the next Collector version can be released.                                                                                                                                 |
| `Sponsor Needed`     | A new component has been proposed, but implementation is not ready to begin. This can be because a sponsor has not yet been decided, or because some details on the component still need to be decided.        |
| `Accepted Component` | A sponsor has elected to take on a component and implementation is ready to begin.                                                                                                                             |
| `Vendor Specific Component` | This should be applied to any component proposal where the functionality for the component is particular to a vendor.                                                                                          |

### State diagram

Here is a diagram outlining the potential issue states and how issues move through different stages:

```mermaid
flowchart TD
    n0(["New issue has been opened"]) --> n1
    n1(["Needs Triage"]) --> n2["Has good repro steps <br>and/or description?"]
  subgraph graph2["**waiting-for-codeowners**"]
        n3["Waiting for Codeowners<br>to further validate the issue"]
  end
  subgraph graph3["**waiting-for-author**"]
        n4["Waiting for author to provide more details"]
  end
  subgraph graph4["**help-wanted**"]
        n8["Waiting on community"]
  end
  subgraph graph5["**closed**"]
        n10(["Close the issue and provide details as needed"])
  end
    n2 -- Yes --> n3
    n2 -- No/Need more details --> n4
    n2 -- Invalid configuration/alternative available --> n10
    n3 -- Invalid Issue --> n10
    n3 -- Valid Issue -->  n6["Codeowner has time<br>to fix it?"]
    n6 -- Assign it to codeowner --> n7["Issue in being worked upon"]
    n6 -- No --> n8
    n7 -- Once PR is merged --> n10
    n8 -- When someone volunteers to provide a fix --> n11["Assign it to the person"]
    n12 -- Any activity on the issue --> n8
    n8 -. Issue becomes stale due to lack of activity .-> n12["Issue is inactive"]
    n11 --> n7
    n12 -- Closed automatically after 120 days due to lack of activity --> n10
    n4 -- Once enough details are available --> n2

    n3@{ shape: rect}
    n4@{ shape: rect}
    n2@{ shape: diam}
    n6@{ shape: diam}

     n1:::Aqua
     n3:::Ash
     n4:::Ash
     n8:::Ash
     n2:::Ash
     n2:::Peach
     n6:::Peach
     n7:::Ash
     n10:::Rose
    classDef Rose stroke-width:1px, stroke-dasharray:none, stroke:#FF5978, fill:#FFDFE5, color:#8E2236
    classDef Aqua stroke-width:1px, stroke-dasharray:none, stroke:#46EDC8, fill:#DEFFF8, color:#378E7A
    classDef Peach stroke-width:1px, stroke-dasharray:none, stroke:#FBB35A, fill:#FFEFDB, color:#8F632D
    classDef Ash stroke-width:1px, stroke-dasharray:none, stroke:#999999, fill:#EEEEEE, color:#000000
```
