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

func test_string_variant_round_trip():
	var msg := OneofOneofHost.new()
	msg.set_text("oneof-string")
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "oneof-string")

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

func test_switching_variants_updates_case():
	# Setting a different field after one is already set should rewrite
	# the active case.
	var msg := OneofOneofHost.new()
	msg.set_number(1)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.NUMBER)
	msg.set_text("now-string")
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.TEXT)
	msg.set_outcome(OneofOutcome.Outcome.OUTCOME_LOSS)
	expect_equal(msg.get_choice_case(), OneofOneofHost.ChoiceOneOf.OUTCOME)

	var decoded := OneofOneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofOneofHost.ChoiceOneOf.OUTCOME)
	expect_equal(decoded.get_outcome(), OneofOutcome.Outcome.OUTCOME_LOSS)
