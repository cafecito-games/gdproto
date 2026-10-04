extends VestTest

# Round-trips the shapes from scoping.proto where a type reference can bind to
# more than one same-named declaration. A wrong binding gives the field one
# declaration's type with another's default value, accessors and codec, so these
# scripts only load at all when the binding is consistent.

func test_nested_message_shadows_top_level_enum_round_trip():
	var msg := ScopingShadowingMessage.new()
	msg.new_field().set_value(7)
	var bytes := msg.to_bytes()

	var decoded := ScopingShadowingMessage.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_field().get_value(), 7)

func test_shadowed_map_value_and_oneof_round_trip():
	var msg := ScopingShadowingMessage.new()
	var entry := ScopingShadowingMessageMode.new()
	entry.set_value(3)
	msg.add_by_name("first", entry)
	var chosen := ScopingShadowingMessageMode.new()
	chosen.set_value(11)
	msg.set_one(chosen)
	var bytes := msg.to_bytes()

	var decoded := ScopingShadowingMessage.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_by_name()["first"].get_value(), 3)
	expect_equal(decoded.get_one().get_value(), 11)
	expect_equal(
		decoded.get_choice_case(),
		ScopingShadowingMessage.ChoiceOneOf.ONE
	)

func test_nested_enum_shadows_top_level_message_round_trip():
	var msg := ScopingShadowingEnum.new()
	msg.set_field(ScopingShadowingEnum.Kind.KIND_NESTED)
	var bytes := msg.to_bytes()

	var decoded := ScopingShadowingEnum.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_field(), ScopingShadowingEnum.Kind.KIND_NESTED)

func test_nested_enum_from_deeper_message_round_trip():
	var msg := ScopingHolderInner.new()
	msg.set_status(ScopingHolder.Status.STATUS_OK)
	var bytes := msg.to_bytes()

	var decoded := ScopingHolderInner.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_status(), ScopingHolder.Status.STATUS_OK)

func test_shadowing_enum_text_uses_the_bound_declarations_value_names():
	var msg := ScopingShadowingLevel.new()
	msg.set_field(ScopingShadowingLevel.Level.LEVEL_NESTED_ONE)
	var text := msg.to_text()
	expect_true(
		text.contains("LEVEL_NESTED_ONE"),
		"text must use the nested enum's value names"
	)
	expect_false(
		text.contains("LEVEL_FILE_SCOPE_ONE"),
		"text must not use the shadowed file-scope enum's value names"
	)

	var decoded := ScopingShadowingLevel.new()
	expect_equal(decoded.from_text(text), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_field(), ScopingShadowingLevel.Level.LEVEL_NESTED_ONE)
