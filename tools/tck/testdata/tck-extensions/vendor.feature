Feature: Vendor extension scenarios

  # This file is not part of the conformance suite. It stands in for what an adopter with
  # provider-specific behaviour writes — flagd's fractional targeting, a vendor's segment rules —
  # and it is here to prove that such a file runs inside the TCK's backend lifecycle rather than
  # beside it.
  #
  # The Background step is the TCK's own, so this scenario shares the provider registration,
  # readiness wait and per-scenario backend reset that the canonical features get. The When and
  # Then steps come from Config.ExtensionSteps.

  Background:
    Given a stable provider

  Scenario: A vendor step resolves a flag through the provider under test
    When the vendor step resolves "string-flag"
    Then the vendor step should have seen "hi"
