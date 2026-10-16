extends VestTest

# Each case carries its default, non-default, and canonical present-default wire.
func cases() -> Array:
	return [
		["int_value", 0, 7, PackedByteArray([8, 0])],
		["string_value", "", "present", PackedByteArray([18, 0])],
		["bool_value", false, true, PackedByteArray([24, 0])],
		["bytes_value", PackedByteArray(), PackedByteArray([65, 66]), PackedByteArray([34, 0])],
		["double_value", 0.0, 1.5, PackedByteArray([41, 0, 0, 0, 0, 0, 0, 0, 0])],
		["float_value", 0.0, 1.5, PackedByteArray([53, 0, 0, 0, 0])],
		["int64_value", 0, 7, PackedByteArray([56, 0])],
		["uint32_value", 0, 7, PackedByteArray([64, 0])],
		["uint64_value", 0, 7, PackedByteArray([72, 0])],
		["sint32_value", 0, 7, PackedByteArray([80, 0])],
		["sint64_value", 0, 7, PackedByteArray([88, 0])],
		["fixed32_value", 0, 7, PackedByteArray([101, 0, 0, 0, 0])],
		["fixed64_value", 0, 7, PackedByteArray([105, 0, 0, 0, 0, 0, 0, 0, 0])],
		["sfixed32_value", 0, 7, PackedByteArray([117, 0, 0, 0, 0])],
		["sfixed64_value", 0, 7, PackedByteArray([121, 0, 0, 0, 0, 0, 0, 0, 0])],
		["choice_value", 0, OptionalChoice.Choice.ONE, PackedByteArray([128, 1, 0])],
	]

func test_optional_presence_all_scalar_families():
	for entry in cases():
		var field: String = entry[0]
		var msg := OptionalOptionalScalars.new()
		expect_equal(msg.call("has_" + field), false)
		expect_equal(msg.call("get_" + field), entry[1])
		expect_equal(msg.to_bytes(), PackedByteArray())
		expect_equal(msg.to_text(), "")
		for value in [entry[1], entry[2]]:
			msg.call("set_" + field, value)
			expect_equal(msg.call("has_" + field), true)
			var wire := msg.to_bytes()
			if value == entry[1]:
				expect_equal(wire, entry[3])
			var decoded := OptionalOptionalScalars.new()
			expect_equal(decoded.from_bytes(wire), ProtoCoreUtils.ProtobufError.NO_ERRORS)
			expect_equal(decoded.call("has_" + field), true)
			expect_equal(decoded.call("get_" + field), value)
			expect_equal(decoded.to_bytes(), wire)
			var text := msg.to_text()
			expect_equal(text.contains(field + ":"), true)
			expect_equal(str(msg).contains(field + ":"), true)
			var parsed := OptionalOptionalScalars.new()
			expect_equal(parsed.from_text(text), ProtoCoreUtils.ProtobufError.NO_ERRORS)
			expect_equal(parsed.call("has_" + field), true)
			expect_equal(parsed.call("get_" + field), value)
			expect_equal(parsed.to_bytes(), wire)
			msg.call("clear_" + field)
			expect_equal(msg.call("has_" + field), false)
			expect_equal(msg.call("get_" + field), entry[1])
			expect_equal(msg.to_bytes(), PackedByteArray())
			expect_equal(msg.to_text(), "")
			expect_equal(str(msg).contains(field + ":"), false)

func test_canonical_present_zero_and_decode_merge():
	var msg := OptionalOptionalScalars.new()
	expect_equal(msg.from_bytes(PackedByteArray([8, 0])), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(msg.has_int_value(), true)
	expect_equal(msg.to_bytes(), PackedByteArray([8, 0]))
	# Existing from_bytes/from_text semantics merge into the receiver.
	expect_equal(msg.from_bytes(PackedByteArray()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(msg.from_text(""), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(msg.has_int_value(), true)
	msg.clear_int_value()
	expect_equal(msg.from_bytes(PackedByteArray()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(msg.has_int_value(), false)

func test_implicit_scalar_defaults_still_elided():
	for entry in cases():
		var field: String = entry[0]
		var msg := OptionalImplicitScalars.new()
		msg.call("set_" + field, entry[1])
		expect_equal(msg.to_bytes(), PackedByteArray())
		expect_equal(msg.to_text(), "")
		expect_equal(msg.has_method("has_" + field), false)
		expect_equal(msg.from_bytes(entry[3]), ProtoCoreUtils.ProtobufError.NO_ERRORS)
		expect_equal(msg.to_bytes(), PackedByteArray())

func test_optional_numeric_enum_text():
	var msg := OptionalOptionalScalars.new()
	expect_equal(msg.from_text("choice_value: 0"), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(msg.has_choice_value(), true)
	expect_equal(msg.get_choice_value(), OptionalChoice.Choice.ZERO)

func test_presence_storage_does_not_collide_with_fields():
	var msg := OptionalPresenceNames.new()
	msg.set_has_count(42)
	msg.set_count(0)
	expect_equal(msg.has_count(), true)
	var decoded := OptionalPresenceNames.new()
	expect_equal(decoded.from_bytes(msg.to_bytes()), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.has_count(), true)
	decoded.clear_count()
	expect_equal(decoded.has_count(), false)
	expect_equal(decoded.get_has_count(), 42)
