# MVP Implementation PLan

## High level plan

**IMPORTANT: NON-NEGOTIABLE**: Before the next step can begin:

- Each phase must pass testing as applicable.
- the user has reviewed and commited the changes.
- The CI build has worked successfully.

| Phase | Step | Goal                                                                                                      | Agent(s)                                |
| ----- | ---- | --------------------------------------------------------------------------------------------------------- | --------------------------------------- |
| 1     | A    | Get the CI build working on Git. That included creating the docker image and pushing it to the ghcr repo. | go devops agent                         |
|       | B    | Update any docs as needed, i.e., README, CHANGELOG, architecture and design docs.                         | go devops agent & documentation agent   |
|       | C    | Implement and unit test the configuration functionality.                                                  | go software agent                       |
|       | D    | Update any docs as needed, i.e., README, CHANGELOG, architecture and design docs.                         | go software agent & documentation agent |
|       | E    | Implement and unit test the observability functionality.                                                  | go software agent & documentation agent |
|       | F    | Update any docs as needed, i.e., README, CHANGELOG, architecture and design docs.                         | go software agent & documentation agent |
