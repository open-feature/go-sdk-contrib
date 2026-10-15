Feature: Flipt vendor behavior

  # Vendor mirror of the canonical @standard-reasons rows
  # (gherkin/reason.feature), which this adoption leaves skipped by
  # withholding tck.StandardReasons: Flipt's evaluation response does not
  # distinguish a rule that matched nothing from a rule-less flag (both are
  # DEFAULT_EVALUATION_REASON), so the provider reports STATIC for both and
  # cannot claim the standard vocabulary. Values and error codes stay covered
  # by the canonical evaluation/errors features; only reasons are mirrored
  # here with Flipt's actual behavior.
  #
  # Names carry a "(Flipt)" prefix so they never collide with the skipped
  # canonical names in reports. None of these scenarios carries a capability
  # tag, so they always run: extension tags gate nothing.
  #
  # Sync note: this file is a manual copy. The digest check in tools/tck
  # covers gherkin/ only, so a spec pin bump that changes reason.feature
  # requires updating these copies by hand (see tools/tck/go.mod pin).

  Background:
    Given a stable provider

  Scenario Outline: (Flipt) A flag with no targeting rules resolves statically
    Given a <type>-flag with key "<key>" and a default value "<default>"
    When the flag was evaluated with details
    Then the reason should be "STATIC"
    And the error-code should be ""
    And no exception should have been thrown

    Examples:
      | key          | type    | default |
      | boolean-flag | Boolean | false   |
      | string-flag  | String  | bye     |
      | integer-flag | Integer | 1       |
      | float-flag   | Float   | 0.1     |

  Scenario: (Flipt) An unknown flag reports an error
    Given a String-flag with key "missing-flag" and a default value "fallback"
    When the flag was evaluated with details
    Then the reason should be "ERROR"
    And the error-code should be "FLAG_NOT_FOUND"
    And no exception should have been thrown

  Scenario: (Flipt) A type mismatch reports an error
    Given a String-flag with key "boolean-flag" and a default value "fallback"
    When the flag was evaluated with details
    Then the reason should be "ERROR"
    And the error-code should be "TYPE_MISMATCH"
    And no exception should have been thrown

  Scenario: (Flipt) A matching targeting rule reports a targeting match
    Given a String-flag with key "targeting-key-flag" and a default value "fallback"
    And a context containing a targeting key with value "5c3d8535-f81a-4478-a6d3-afaa4d51199e"
    When the flag was evaluated with details
    Then the reason should be "TARGETING_MATCH"
    And the error-code should be ""
    And no exception should have been thrown

  Scenario: A targeting rule that does not match resolves statically
    # Canonical expects DEFAULT here; Flipt answers the same DEFAULT
    # evaluation reason as a rule-less flag, so the provider reports STATIC.
    Given a String-flag with key "targeting-key-flag" and a default value "fallback"
    And a context containing a targeting key with value "f20bd32d-703b-48b6-bc8e-79d53c85134a"
    When the flag was evaluated with details
    Then the reason should be "STATIC"
    And the error-code should be ""
    And no exception should have been thrown

  Scenario: (Flipt) A disabled flag reports that it is disabled
    Given a Boolean-flag with key "disabled-boolean-flag" and a default value "false"
    When the flag was evaluated with details
    Then the reason should be "DISABLED"
    And the error-code should be ""
    And no exception should have been thrown

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
