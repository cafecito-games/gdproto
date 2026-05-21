extends VestTest

func test_unset_default_case():
	var msg := OneofProto.OneofHost.new()
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.UNSET)

	var bytes := msg.to_bytes()
	expect_equal(bytes.size(), 0)
	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.UNSET)

func test_scalar_variant_round_trip():
	var msg := OneofProto.OneofHost.new()
	msg.set_number(42)
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.NUMBER)

	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.NUMBER)
	expect_equal(decoded.get_number(), 42)

func test_string_variant_round_trip():
	var msg := OneofProto.OneofHost.new()
	msg.set_text("oneof-string")
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.TEXT)

	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.TEXT)
	expect_equal(decoded.get_text(), "oneof-string")

func test_message_variant_round_trip():
	var msg := OneofProto.OneofHost.new()
	var payload := OneofProto.OneofPayload.new()
	payload.set_note("inside")
	msg.set_payload(payload)
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.PAYLOAD)

	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.PAYLOAD)
	expect_equal(decoded.get_payload().get_note(), "inside")

func test_enum_variant_round_trip():
	var msg := OneofProto.OneofHost.new()
	msg.set_outcome(OneofProto.Outcome.OUTCOME_WIN)
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.OUTCOME)

	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.OUTCOME)
	expect_equal(decoded.get_outcome(), OneofProto.Outcome.OUTCOME_WIN)

func test_switching_variants_updates_case():
	# Setting a different field after one is already set should rewrite
	# the active case.
	var msg := OneofProto.OneofHost.new()
	msg.set_number(1)
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.NUMBER)
	msg.set_text("now-string")
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.TEXT)
	msg.set_outcome(OneofProto.Outcome.OUTCOME_LOSS)
	expect_equal(msg.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.OUTCOME)

	var decoded := OneofProto.OneofHost.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_choice_case(), OneofProto.OneofHost.ChoiceOneOf.OUTCOME)
	expect_equal(decoded.get_outcome(), OneofProto.Outcome.OUTCOME_LOSS)
