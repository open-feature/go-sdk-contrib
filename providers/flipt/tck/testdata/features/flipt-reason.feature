Feature: Flipt vendor behavior

  # Flipt-specific behavior the canonical suite cannot assert.
  #
  # Two canonical rows are disabled by name (see WithKnownDeviations) and their
  # Flipt behavior is asserted here instead: a targeting rule that matched
  # nothing reports the same DEFAULT evaluation reason as a rule-less flag, so
  # the provider reports STATIC where the canonical suite expects DEFAULT; and
  # the canonical falsy scenario expects the empty string, but Flipt forbids
  # empty variant keys, so the seed carries "zero" instead.
  #
  # Everything else canonical runs unmodified, so nothing here repeats it. The
  # disabled falsy outline shares one scenario name across its rows, so
  # disabling it also skips the passing boolean/integer rows; those are
  # re-asserted below.
  #
  # None of these scenarios carries a capability tag, so they always run. Their
  # names must not collide with the disabled list, which matches by name across
  # the whole run: a vendor scenario sharing a disabled name would skip itself.

  Background:
    Given a stable provider

  Scenario: A flag with no default variant reports the default reason
    # no-default-flag is hand-seeded with no default variant, so Flipt answers
    # UNKNOWN with no match and the provider falls back to the caller's default
    # with reason DEFAULT instead of reporting a type mismatch on the empty key.
    Given a String-flag with key "no-default-flag" and a default value "fallback"
    When the flag was evaluated with details
    Then the resolved details value should be "fallback"
    And the reason should be "DEFAULT"
    And the error-code should be ""
    And no exception should have been thrown

  Scenario: A targeting rule that does not match resolves statically
    Given a String-flag with key "targeting-key-flag" and a default value "fallback"
    And a context containing a targeting key with value "f20bd32d-703b-48b6-bc8e-79d53c85134a"
    When the flag was evaluated with details
    Then the reason should be "STATIC"
    And the error-code should be ""
    And no exception should have been thrown

  Scenario: Flipt string-zero-flag resolves to its seeded key
    Given a String-flag with key "string-zero-flag" and a default value "fallback"
    When the flag was evaluated with details
    Then the resolved details value should be "zero"
    And the error-code should be ""
    And the error message should be empty
    And no exception should have been thrown

  Scenario Outline: Falsy values are values, not absences
    # The boolean/integer rows of the disabled canonical falsy outline,
    # re-asserted so disabling the shared outline name loses no coverage.
    Given a <type>-flag with key "<key>" and a default value "<default>"
    When the flag was evaluated with details
    Then the resolved details value should be "<value>"
    And the error-code should be ""
    And the error message should be empty
    And no exception should have been thrown

    Examples:
      | key               | type    | default | value |
      | boolean-zero-flag | Boolean | true    | false |
      | integer-zero-flag | Integer | 1       | 0     |
