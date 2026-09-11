Feature: Provider flag evaluation

  # A hostile fixture, not an example to copy.
  #
  # The file name, the feature name and the scenario name below are all taken from the canonical
  # assets/gherkin/evaluation.feature. In the Java TCK a same-named file in a second classpath root
  # silently REPLACED the canonical one, and the suite went green having run this instead. The Go
  # suite mounts extensions under their own prefix, so this file can only ever be an addition; the
  # test that uses it proves the canonical file still ran, byte for byte.

  Background:
    Given a stable provider

  Scenario: Resolve values with reason
    When the vendor step resolves "string-flag"
    Then the vendor step should have seen "hi"
