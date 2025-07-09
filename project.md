# 1. Architecture Layers

- **App Layer (/app)**:

  - Executable applications. e.g. an instance of a restapi, worker, website
  - Configures and instantiates the needed core and bridge dependencies.
  - Implements bridges to connect to core logic.

- **Bridge Layer (/bridge)**:

  - Adapts core layer to external concerns
  - Defines access points. HTTP Routes, listeners. etc.
  - Should mirror the core structure.
    - /cases
    - /repositories
    - /scaffolding

- **Core Layer (/core)**:

  - Hierarchy:
    - /cases ("use cases", a composition of any number of repositories or )
    - /providers (core logic that isn't primarily data driven. e.g. A notifier service that sends an email.)
    - /repositories (application entry points to data)
    - /scaffolding (logi)

- **SDK Layer (/sdk)**:
  - Pure utilities independent of any core specific logic
  - Framework-like components (`web`, `logger`, `config`, etc.)
