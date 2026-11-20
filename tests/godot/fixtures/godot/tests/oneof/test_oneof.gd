extends VestTest

func test_unset_default_case():
	var msg := OneofOneofHost.new()
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.UNSET)

	var bytes := msg.to_bytes()
	expect_equal(bytes.size(), 0)
	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.UNSET)

func test_scalar_variant_round_trip():
	var msg := OneofOneofHost.new()
	msg.set_number(42)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.NUMBER)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.NUMBER)
	expect_equal(decoded.get_number(), 42)

	var text_decoded := OneofOneofHost.new()
	expect_equal(text_decoded.from_text(msg.to_text()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(text_decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.NUMBER)
	expect_equal(text_decoded.get_number(), 42)

func test_string_variant_round_trip():
	var msg := OneofOneofHost.new()
	msg.set_text("oneof-string")
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "oneof-string")

	var text_decoded := OneofOneofHost.new()
	expect_equal(text_decoded.from_text(msg.to_text()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(text_decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(text_decoded.get_text(), "oneof-string")

func test_message_variant_round_trip():
	var msg := OneofOneofHost.new()
	var payload := OneofOneofPayload.new()
	payload.set_note("inside")
	msg.set_payload(payload)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.PAYLOAD)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.PAYLOAD)
	expect_equal(decoded.get_payload().get_note(), "inside")

func test_enum_variant_round_trip():
	var msg := OneofOneofHost.new()
	msg.set_outcome(OneofOutcome.Outcome.OUTCOME_WIN)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.OUTCOME)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.OUTCOME)
	expect_equal(decoded.get_outcome(), OneofOutcome.Outcome.OUTCOME_WIN)

func test_switching_all_categories_resets_previous_arms():
	var msg := OneofOneofHost.new()
	msg.set_number(7)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.NUMBER)
	expect_equal(msg.get_number(), 7)

	msg.set_flag(true)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.FLAG)
	expect_equal(msg.get_number(), 0)
	expect_equal(msg.get_flag(), true)

	msg.set_ratio(3.5)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.RATIO)
	expect_equal(msg.get_flag(), false)
	expect_equal(is_equal_approx(msg.get_ratio(), 3.5), true)

	msg.set_raw(PackedByteArray([1, 2, 3]))
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.RAW)
	expect_equal(msg.get_ratio(), 0.0)
	expect_equal(msg.get_raw(), PackedByteArray([1, 2, 3]))

	var payload := OneofOneofPayload.new()
	payload.set_note("nested")
	msg.set_payload(payload)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.PAYLOAD)
	expect_equal(msg.get_raw().size(), 0)
	expect_equal(msg.get_payload().get_note(), "nested")

	msg.set_outcome(OneofOutcome.Outcome.OUTCOME_LOSS)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.OUTCOME)
	expect_equal(msg.get_payload(), null)
	expect_equal(msg.get_outcome(), OneofOutcome.Outcome.OUTCOME_LOSS)

	msg.set_text("final")
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(msg.get_outcome(), OneofOutcome.Outcome.OUTCOME_UNSPECIFIED)
	expect_equal(msg.get_text(), "final")

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "final")

func test_stale_decoded_multi_arm_state_is_cleared_on_case_transition():
	# Concatenating two separate oneof encodings creates stale backing state
	# on the earlier arm because binary decoding writes fields directly.
	var first := OneofOneofHost.new()
	first.set_number(99)
	var second := OneofOneofHost.new()
	second.set_text("second")

	var combined := PackedByteArray()
	combined.append_array(first.to_bytes())
	combined.append_array(second.to_bytes())

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(combined), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "second")
	expect_equal(decoded.get_number(), 99)

	# Same-case write preserves current same-case semantics (does not clear
	# inactive backing state).
	decoded.set_text("updated")
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "updated")
	expect_equal(decoded.get_number(), 99)

	# Switching to another case clears every backing arm, including stale ones.
	decoded.set_flag(true)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.FLAG)
	expect_equal(decoded.get_flag(), true)
	expect_equal(decoded.get_number(), 0)
	expect_equal(decoded.get_text(), "")

func test_multiple_oneof_groups_remain_independent():
	var msg := OneofDualOneofHost.new()
	msg.set_first_number(11)
	msg.set_second_flag(true)
	expect_equal(msg.get_primary_case(), OneofDualOneofHost.PrimaryOneOf.FIRST_NUMBER)
	expect_equal(msg.get_secondary_case(), OneofDualOneofHost.SecondaryOneOf.SECOND_FLAG)

	msg.set_first_text("switched")
	expect_equal(msg.get_primary_case(), OneofDualOneofHost.PrimaryOneOf.FIRST_TEXT)
	expect_equal(msg.get_first_number(), 0)
	expect_equal(msg.get_first_text(), "switched")
	expect_equal(msg.get_secondary_case(), OneofDualOneofHost.SecondaryOneOf.SECOND_FLAG)
	expect_equal(msg.get_second_flag(), true)

	msg.set_second_ratio(2.25)
	expect_equal(msg.get_secondary_case(), OneofDualOneofHost.SecondaryOneOf.SECOND_RATIO)
	expect_equal(msg.get_second_flag(), false)
	expect_equal(is_equal_approx(msg.get_second_ratio(), 2.25), true)
	expect_equal(msg.get_primary_case(), OneofDualOneofHost.PrimaryOneOf.FIRST_TEXT)
	expect_equal(msg.get_first_text(), "switched")

func test_collision_named_field_and_oneof_helper_coexist():
	var msg := OneofCollideOneofHost.new()
	msg.set_select_choice("keep-me")
	msg.set_number(5)
	expect_equal(msg.get_select_choice(), "keep-me")
	expect_equal(msg.get_choice_case(), OneofCollideOneofHost.ChoiceOneOf.NUMBER)
	expect_equal(msg.get_number(), 5)

	msg.set_text("switched")
	expect_equal(msg.get_select_choice(), "keep-me")
	expect_equal(msg.get_choice_case(), OneofCollideOneofHost.ChoiceOneOf.TEXT)
	expect_equal(msg.get_number(), 0)
	expect_equal(msg.get_text(), "switched")
