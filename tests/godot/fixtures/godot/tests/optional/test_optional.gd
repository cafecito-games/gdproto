extends VestTest

# The generator currently emits proto3-optional fields as regular scalars
# without presence tracking (no has_*/clear_* helpers, default elision on
# write). These tests pin down that working subset so a future change that
# adds explicit-presence support has a regression boundary, and so the
# scalar round-trip path keeps working today.

func test_optional_scalars_round_trip_binary():
	var msg := OptionalOptionalScalars.new()
	msg.set_int_value(7)
	msg.set_string_value("present")
	msg.set_bool_value(true)

	var bytes := msg.to_bytes()
	var decoded := OptionalOptionalScalars.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_int_value(), 7)
	expect_equal(decoded.get_string_value(), "present")
	expect_equal(decoded.get_bool_value(), true)

func test_optional_unset_round_trip_binary():
	var msg := OptionalOptionalScalars.new()
	var bytes := msg.to_bytes()
	# Without explicit-presence support we elide unset/default scalars.
	expect_equal(bytes.size(), 0)

	var decoded := OptionalOptionalScalars.new()
	expect_equal(decoded.from_bytes(bytes), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_int_value(), 0)
	expect_equal(decoded.get_string_value(), "")
	expect_equal(decoded.get_bool_value(), false)

func test_optional_round_trip_text():
	var msg := OptionalOptionalScalars.new()
	msg.set_int_value(99)
	msg.set_string_value("text")
	var text := msg.to_text()

	var decoded := OptionalOptionalScalars.new()
	expect_equal(decoded.from_text(text), ProtoCoreUtils.ProtobufError.NO_ERRORS)
	expect_equal(decoded.get_int_value(), 99)
	expect_equal(decoded.get_string_value(), "text")
