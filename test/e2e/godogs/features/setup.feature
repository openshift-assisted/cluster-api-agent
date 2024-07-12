Feature: Setting up workloads on a Kubernetes cluster
    In order to run my tests
    I need to have a working Kubernetes cluster
    With the required workloads successfully running

    Scenario: Kubernetes cluster exists and no workloads are installed
        Given a working Kubernetes cluster
        When no workloads are installed
        Then I want to install all workloads on the Kubernetes cluster
        And check that they're all successfully running
